package main

import "fmt"
import "os"
import "bytes"
import "encoding/binary"
import "strings"
import "flag"
import "database/sql"
import _ "modernc.org/sqlite"	
import "MFS2SQL/db"
import "MFS2SQL/internal"
import "MFS2SQL/parser"
import "MFS2SQL/intro"

func iterateMFT(driveLocation string, mftBlockOffset uint64, recordSize int64, ignoreRecords int, NTFSOffset uint32, clusterSize uint32, outputMode int) int{
	// Note that the first 26 records are reserved for system specific purposes: http://ntfs.com/ntfs-system-files.htm
	// But this only holds for the first block
	handle, error := os.Open(driveLocation)
	fileIndicator := [4]byte{70, 73, 76, 69}		// Note, this spells out FILE, based on the decimal values for the corresponding character in the ASCII table.
	var tmpMagicNumber [4]byte
	recordCounter := int64(ignoreRecords)			// the firest 26 contain some unsued ones, will mess up the loop.
	if(error == nil){
		// Initialize first record
		recordOffset := int64(mftBlockOffset) + int64((recordCounter * recordSize))
		mftRecordBuffer := make([]byte, recordSize)
		handle.Seek(recordOffset,0)		
		handle.Read(mftRecordBuffer)
		binary.Read(bytes.NewBuffer(mftRecordBuffer[0:4]), binary.LittleEndian, &tmpMagicNumber)

		for(tmpMagicNumber == fileIndicator){
			// Load next record
			recordCounter +=1
			recordOffset := int64(mftBlockOffset) + int64((recordCounter * recordSize))
			handle.Seek(recordOffset,0)
			handle.Read(mftRecordBuffer)
			processFileRecord(parser.ParseMFTRecord(mftRecordBuffer, recordOffset, NTFSOffset, clusterSize, outputMode), outputMode)
			binary.Read(bytes.NewBuffer(mftRecordBuffer[0:4]), binary.LittleEndian, &tmpMagicNumber)
		}
	}
	handle.Close()
	return int(recordCounter)
}


// *** Support drive specific
func identifyBasicPartition(partitionArray []internal.PARTITIONENTRY, basicPartionGUID [16]byte) (bool, int){
	for partitionNumber, partition := range partitionArray{
		if partition.PartitionGUID == basicPartionGUID{
			return true, partitionNumber
		}
	}
	return false,0
}

// *** User functionality *** /
/* Output modus: 1 = Write to screen, 2=Create SQL DB*/
func processFileRecord(fileInformation internal.FILE_INFO, outputMode int){
	if(outputMode == 1){
		fmt.Printf("Finished Filename: %s. \n isActive: %t \n isFolder: %t \nStarting at: %d with size: %d\nParent directory: :d", fileInformation.FileName, fileInformation.IsActive, fileInformation.IsFolder,fileInformation.FullDataOffset,fileInformation.DataLength, fileInformation.ParentDirectory)
	}
	if(outputMode == 2){
		db.InsertFileRecord(int(fileInformation.RecordID), fileInformation.FileName, int(fileInformation.ParentDirectory), internal.BoolToInt(fileInformation.IsFolder), internal.BoolToInt(fileInformation.IsActive), int(fileInformation.FullDataOffset), int(fileInformation.DataLength))
	}
}


/* Carve functionality */
func dumpToFile(deviceLocation string, offset int, length int, outputFile string){
	fmt.Println("[+] Dumping file with offset: ", offset, " length: ", length, " into file: ", outputFile)
	physicalDiskHandle, _ := os.Open(deviceLocation)
	// Note: buffer needs to be a multiple of 512 to work.
	
	buffer := make([]byte, int64(length))
	physicalDiskHandle.Seek(int64(offset),0)
	physicalDiskHandle.Read(buffer)
	os.WriteFile(outputFile, buffer, 0644)
	physicalDiskHandle.Close()
}

/* MFT to DB or File functionality */
func dumpMFT(deviceLocation string, dumpMode int){
	const logicalBlockAddressSize = 512
	const GPTLBA = 1									//we need the first LBA, LBA0 is legacy	
	
	// Typical GUIDs: https://learn.microsoft.com/en-us/windows/win32/api/winioctl/ns-winioctl-partition_information_gpt
	// We are searching for: ebd0a0a2-b9e5-4433-87c0-68b6b72699c7
	NTFSSearchGUID := [16]byte{162, 160, 208, 235, 229, 185, 51, 68, 135, 192, 104, 182, 183, 38, 153, 199} 
	NTFSOEMIndicator := [8]byte{78, 84, 70, 83, 32, 32, 32, 32}
	const NTFSBootSectorSize = 512
	const recordSize = 1024
	totalRecords := 0
	
	fmt.Println("[+] Parsing GPT Header")
	gptheader := parser.ParseGPTHeader(deviceLocation, logicalBlockAddressSize, GPTLBA)				
	fmt.Printf("[+] Calculating buffer size for DISK with signature % x", gptheader.Signature)
	partitionTableOffset := gptheader.PartitionEntriesLBA * logicalBlockAddressSize
	fmt.Printf("\n  --> Starting at LBA: %d means a seek offset of: %d", gptheader.PartitionEntriesLBA, partitionTableOffset)
	fmt.Printf("\n  --> With %d partitions, of size: %d, we need a buffer of: %d", gptheader.NumberOfPartitions, gptheader.PartitionEntrySize, gptheader.NumberOfPartitions * gptheader.PartitionEntrySize)
	fmt.Println("\n[+] Parsing Partition table")
	numberOfPartitions, partitions := parser.ParsePartitions(deviceLocation, partitionTableOffset, gptheader.PartitionEntrySize, gptheader.NumberOfPartitions)
	fmt.Printf("  --> Number of partitions identified: %d",numberOfPartitions)
	fmt.Println("\n[+] Determining windows Base partition / NTFS partition")
	basicPartitionFound, partNumber := identifyBasicPartition(partitions, NTFSSearchGUID)
	if(basicPartitionFound){
		NTFSOffset := logicalBlockAddressSize * partitions[partNumber].StartingLBA
		fmt.Printf("  --> Found basic partition starting at offset: %d",NTFSOffset)
		fmt.Println("\n[+] Parsing NTFS header")
		ntfsHeader := parser.ParseNTFSHeader(deviceLocation,NTFSOffset, NTFSBootSectorSize)
		if(ntfsHeader.OemID == NTFSOEMIndicator){
			fmt.Println("  --> Validated basic partition to be NTFS by comparing oemID")
			fmt.Printf("  --> Using BytesPerSector: %d, SectorsPerCluster: %d\n",ntfsHeader.BytesPerSector, ntfsHeader.SectorPerCluster)
			clusterSize := uint32(ntfsHeader.BytesPerSector)*uint32(ntfsHeader.SectorPerCluster)
			MFTOffset := NTFSOffset + ntfsHeader.MFTOffset*clusterSize
			fmt.Printf("  --> Master File Table ($MFT) offset found at: %d, e.g. a total offset of: %d", ntfsHeader.MFTOffset, MFTOffset)
			fmt.Printf("\n  --> $MFT offset - NFTSoffset (as used in the table): %d or %x in hex", MFTOffset - NTFSOffset,MFTOffset - NTFSOffset)
			fmt.Println("\n[+] Parsing Master File Table (this can take a while)")
			MFTBlockArray := parser.GetMFTOffsetLocationsFromMFT(deviceLocation, MFTOffset, recordSize, NTFSOffset)
			fmt.Printf("  --> Found %d MFT Blocks\n\n", len(MFTBlockArray))
			// The first MFT Block, contains the $MFT file as well. The first 26 files (include the $MFT file, $MFT mirror, etc.) also have some slack ones. Hence we skip parsing them for the sake of simplicity
			totalRecords = iterateMFT(deviceLocation, uint64(MFTBlockArray[0]), recordSize, 26, NTFSOffset, clusterSize, dumpMode)
			for blockIndex := 1; blockIndex < len(MFTBlockArray); blockIndex++ {
				totalRecords = totalRecords + iterateMFT(deviceLocation, uint64(MFTBlockArray[blockIndex]), recordSize, 0, NTFSOffset, clusterSize, dumpMode)
			}
			// Flush DB insert, just in case any records are still left in memory
			db.FlushBatch()
			
			fmt.Printf("\n  --> Found %d files in the $MFT records",totalRecords)
		}
	}
}

// search sql database, for the file, and print info
func searchFileAndPrintInfo(userInput string, dbFile string) bool{
	// Set-up our DB connection
    var err error
	var database *sql.DB
    database, err = sql.Open("sqlite", dbFile)
    if err != nil {
        fmt.Println("[!] Error opening database:", err)
        return false
    }

    // Ensure the DB connection is alive
    if err = database.Ping(); err != nil {
        fmt.Println("[!] Failed to connect to database:", err)
        return false
    }
	
	// Fix user input (remove Drive letter,. remove escaping, isn't needed, abort if no file is provided)
	userInput = strings.ReplaceAll(userInput, "//./", "")
	userInput = strings.ReplaceAll(userInput, "//", "/")

	lastSep := strings.LastIndex(userInput, `\`)
    if lastSep == -1 {
        fmt.Println("[!] Invalid path format")
        return false
    }
    file := userInput[lastSep+1:]
    path := userInput

    // Remove drive letter (e.g. "C:\"), user shouldn't input this, but regardless kill it, if its there
    if colonIdx := strings.Index(path, `:\`); colonIdx != -1 {
        path = path[colonIdx+2:]
    }
	
	query := "SELECT RID, fileOffset, fileLength, isActive FROM files WHERE filename = ? AND fullPath = ? COLLATE NOCASE"

    row := database.QueryRow(query, file, path)
    var rid, offset, length, active int
    err = row.Scan(&rid, &offset, &length, &active)
    if err != nil {
        fmt.Println("[!] No matching entry found:", err)
        return false
    }

    fmt.Println("📄 File:", file)
    fmt.Println("Offset:", offset)
    fmt.Println("Length:", length)
	fmt.Println("Command: go run MFT2SQL.go -carve -fileOffset ", offset, " -fileLength ", length)

	return true
}







func runModeDispatcher(help bool, carve bool, getFileLocation string, dumpMode int, deviceLocation string, fileOffset int, fileLength int, dumpFile string, dbFile string) {
    // Default behavior: show help banner
    if help || (!carve && getFileLocation == "" && dumpMode == 0) {
        intro.ShowBannerAndIntro()
        flag.Usage()
        os.Exit(0)
    }

    if carve {
        if fileOffset == 0 || fileLength == 0 {
            fmt.Println("[!] Please provide both fileOffset and fileLength when using --carve.")
            return
        }
        fmt.Println("[+] Carving file from disk...")
        dumpToFile(deviceLocation, fileOffset, fileLength, dumpFile)
        return
    }

    if getFileLocation != "" {
        fmt.Println("[+] Fetching file location info for:", getFileLocation)
		if(!searchFileAndPrintInfo(getFileLocation, dbFile)){
			os.Exit(1)
		}
        return
    }

    if dumpMode == 2 {
        if !db.SetUpSQLiteDB(dbFile) {
            fmt.Println("[+] Could not initialize the database. Exiting.")
            os.Exit(1)
        }
        db.InsertCounter = 0
        dumpMFT(deviceLocation, dumpMode)
        db.UpdateFullpaths()
        return
    }

    if dumpMode == 1 {
        fmt.Println("[+️] Dumping MFT entries to screen...")
        dumpMFT(deviceLocation, dumpMode)
        return
    }

    fmt.Println("[-] No valid mode selected. Try using -help for usage.")
}

func main() {
    var help = flag.Bool("help", false, "Show help banner and usage.")
    var deviceLocation = "\\\\.\\physicaldrive0"
    var dumpMode int
    var carve = false
    var fileOffset int
    var fileLength int
    var getFileLocation = ""
    var dbFile = "MFTDB.db"
    var dumpFile = "output.dump"

    flag.StringVar(&deviceLocation, "deviceLocation", deviceLocation, "Specify the physical disk to dump")
    flag.IntVar(&dumpMode, "dumpMode", 0, "Select MFT dump output: 1=screen, 2=SQL")
    flag.StringVar(&dbFile, "dbFile", dbFile, "Specify the name of the SQLite database")
    flag.StringVar(&dumpFile, "dumpFile", dumpFile, "Output file name for carving")
    flag.StringVar(&getFileLocation, "getFileLocation", getFileLocation, "Lookup file location using its full path")
	flag.BoolVar(&carve, "carve", carve, "Carve a file from disk, make sure -fileOffset and -fileLength are provided")
    flag.IntVar(&fileOffset, "fileOffset", fileOffset, "Offset to start carving file from physical disk")
    flag.IntVar(&fileLength, "fileLength", fileLength, "Length of file to carve")

    flag.Parse()

	// Application requires administrator privileges
	if(!internal.IsAdmin()){
		fmt.Println("[!] This tool must be run with administrative privileges.")
        os.Exit(1)
	} else{
		runModeDispatcher(*help, carve, getFileLocation, dumpMode, deviceLocation, fileOffset, fileLength, dumpFile, dbFile)
	}
}


