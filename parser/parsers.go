package parser

import "bytes"
import "encoding/binary"
import "fmt"
import "MFS2SQL/internal"
import "strconv"
import "os"

func interPreteMFTRecordFlag(flag uint16)(bool,bool){
	// https://flatcap.github.io/linux-ntfs/ntfs/concepts/file_record.html
	isFolder := false
	isActive := false //e.g. file is deleted
	
	if flag >= 8{
		// Special index flag, not relevant for this implementation
		flag = flag - 8
	}
	if flag >= 4{
		// Extension flag, not relevant to this implementation
		flag = flag - 4
	}
	if flag >= 2{
		flag = flag - 2
		isFolder = true
	}
	if flag >= 1{
		isActive = true
	}
	return isFolder, isActive
}


func getNumber(input []byte) int64{
	var returnNumber int64
	var offsetAsString string
	
	if(len(input) > 4){
		fmt.Println("Error: out of range")
	}
	if(len(input) == 4){
		offsetAsString = offsetAsString + fmt.Sprintf("%x",input[3])
		offsetAsString = offsetAsString + fmt.Sprintf("%x",input[2])
		offsetAsString = offsetAsString + fmt.Sprintf("%x",input[1])
		offsetAsString = offsetAsString + fmt.Sprintf("%x",input[0])		
	}
	if(len(input) == 3){
		offsetAsString = offsetAsString + fmt.Sprintf("%x",input[2])
		offsetAsString = offsetAsString + fmt.Sprintf("%x",input[1])
		offsetAsString = offsetAsString + fmt.Sprintf("%x",input[0])		
	}
	
	if(len(input)==2){
		offsetAsString = offsetAsString + fmt.Sprintf("%x",input[1])
		offsetAsString = offsetAsString + fmt.Sprintf("%x",input[0])			
	}
	
	if(len(input)==1){
		offsetAsString = offsetAsString + fmt.Sprintf("%x",input[0])			
	}
	returnNumber,_ = strconv.ParseInt(offsetAsString, 16, 64)
	return returnNumber
}


func getFilenameAsString(fileNameLength uint8, offset uint8, attribute []byte)(string){	
	fileName := ""
	fileNameLength2 := fileNameLength*2
	start := offset + 66
	stop := uint16(start) + uint16(fileNameLength2)
	fileNameWithZeros := attribute[start:stop]
				
	for _, character := range fileNameWithZeros{
		if character != 0{
			fileName = fileName + string(character)
		}
	}
	return fileName
}

func ParseMFTRecord(recordBuffer []byte, recordOffset int64, NTFSOffset uint32, clusterSize uint32, outputMode int) internal.FILE_INFO{
	fileIndicator := [4]byte{70, 73, 76, 69}	// The numbers correspond to the FILE characters
	var tmpMagicNumber [4]byte
	binary.Read(bytes.NewBuffer(recordBuffer[0:4]), binary.LittleEndian, &tmpMagicNumber)
	var fileInformation internal.FILE_INFO
	
	if(tmpMagicNumber == fileIndicator){
		
		
		var offsetToAttribute uint16
		var fileRecordFlag uint16
		var sizeOfRecord uint32
		var checkEndMarker uint32
		
		endMarker := uint32(4294967295)	// the filerecord ends with end marker: 0xFFFFFFFF (e.g. 4294967295 in dec)
		// Default attribute information: https://flatcap.github.io/linux-ntfs/ntfs/concepts/attribute_header.html
		binary.Read(bytes.NewBuffer(recordBuffer[20:22]), binary.LittleEndian, &offsetToAttribute)		// This usually becomes 0x38 or 56 for the first attribute
		binary.Read(bytes.NewBuffer(recordBuffer[22:24]), binary.LittleEndian, &fileRecordFlag)
		binary.Read(bytes.NewBuffer(recordBuffer[24:28]), binary.LittleEndian, &sizeOfRecord)
		binary.Read(bytes.NewBuffer(recordBuffer[44:48]), binary.LittleEndian, &fileInformation.RecordID)

		fileInformation.IsFolder, fileInformation.IsActive = interPreteMFTRecordFlag(fileRecordFlag)

		// The first 4 bytes of an attribute are the attribute type, the second 4 bytes are the attribute length		
		for(checkEndMarker != endMarker){
			var attributeType uint16
			var attributeLength uint16
			binary.Read(bytes.NewBuffer(recordBuffer[offsetToAttribute:offsetToAttribute +4]), binary.LittleEndian, &attributeType)
			binary.Read(bytes.NewBuffer(recordBuffer[offsetToAttribute+4:offsetToAttribute +8]), binary.LittleEndian, &attributeLength)
			// If there are no attributes, you will read FFFF or 65535 as attribute type (which is the end marker)
			if(attributeType != 65535){
				// Check for our attributes, we want $DATA (0x80 or 128) and $FILE_NAME (0x30 or 48)
				// Full list for anybody that wants to complete this implementation: https://learn.microsoft.com/en-us/windows/win32/devnotes/attribute-list-entry
				attribute := recordBuffer[offsetToAttribute:(offsetToAttribute + attributeLength)]
				// attribute 0x10 contains the Standard information, which is kept up to date: https://flatcap.github.io/linux-ntfs/ntfs/attributes/file_name.html
				if(attributeType == 16){
					var ofssetToAttributeData uint8		
					binary.Read(bytes.NewBuffer(attribute[20:22]), binary.LittleEndian, &ofssetToAttributeData)
				
					// To do: implement time conversaion: https://pkg.go.dev/google.golang.org/protobuf/types/known/timestamppb
					binary.Read(bytes.NewBuffer(attribute[ofssetToAttributeData:ofssetToAttributeData + 8]), binary.LittleEndian, &fileInformation.FileCreatedUTCWinFileEpoch)
					binary.Read(bytes.NewBuffer(attribute[ofssetToAttributeData + 8:ofssetToAttributeData + 16]), binary.LittleEndian, &fileInformation.FileModifiedUTCWinFileEpoch)
					binary.Read(bytes.NewBuffer(attribute[ofssetToAttributeData + 16:ofssetToAttributeData + 24]), binary.LittleEndian, &fileInformation.FileRecordModifiedUTCWinFileEpoch)
					binary.Read(bytes.NewBuffer(attribute[ofssetToAttributeData + 24:ofssetToAttributeData + 32]), binary.LittleEndian, &fileInformation.FileLastReadUTCWinFileEpoch)
				
					// To do: write parser for flag to optimize DB queriying for insecure permission combinations
					binary.Read(bytes.NewBuffer(attribute[ofssetToAttributeData + 32:ofssetToAttributeData + 40]), binary.LittleEndian, &fileInformation.FilePermissionFlag)
					binary.Read(bytes.NewBuffer(attribute[ofssetToAttributeData + 48:ofssetToAttributeData + 52]), binary.LittleEndian, &fileInformation.FileOwnerID)
				}
			
				// attribute 0x30 contains the information attribute, including the filename
				if(attributeType == 48){
					var ofssetToAttributeData uint8		
					var fileNameLength uint8		
				
					binary.Read(bytes.NewBuffer(attribute[20:22]), binary.LittleEndian, &ofssetToAttributeData)
					binary.Read(bytes.NewBuffer(attribute[ofssetToAttributeData:ofssetToAttributeData + 6]), binary.LittleEndian, &fileInformation.ParentDirectory)
					binary.Read(bytes.NewBuffer(attribute[ofssetToAttributeData +64:ofssetToAttributeData + 65]), binary.LittleEndian, &fileNameLength)
					fileInformation.FileName = getFilenameAsString(fileNameLength, ofssetToAttributeData, attribute)
				}
			
				// attribute 0x80 contains the data offset, more information about the data attributes: https://sabercomlogica.com/en/ntfs-non-resident-and-no-named-attributes/
				if(attributeType == 128){
					// Some limitations, a file record can have multiple $DATA sections. Additionally, a non-resident data record, can have multiple data runs.
					var noneResidentFlag uint8
					var ofssetToAttributeData uint16
					binary.Read(bytes.NewBuffer(attribute[8:9]), binary.LittleEndian, &noneResidentFlag)

					// Data in file record
					if noneResidentFlag == 0{
						binary.Read(bytes.NewBuffer(attribute[16:20]), binary.LittleEndian, &fileInformation.DataLength)
						binary.Read(bytes.NewBuffer(attribute[20:22]), binary.LittleEndian, &ofssetToAttributeData)
						// To do calculate this back, to get absolute offset (note that the record offset, also includes the NTFS offset)
						fileInformation.FullDataOffset = uint64(offsetToAttribute) + uint64(ofssetToAttributeData) + uint64(recordOffset)
					}
				
					// Data outside
					if noneResidentFlag == 1{
						binary.Read(bytes.NewBuffer(attribute[48:56]), binary.LittleEndian, &fileInformation.DataLength)
						binary.Read(bytes.NewBuffer(attribute[32:34]), binary.LittleEndian, &ofssetToAttributeData)
						// Bold move, i'm not going to care for large files with multiple data runs, if you need those, make your own implementation :)
						// To save space, the dataRun varies in space, the only thing we know for sure is that the length is the first byte: http://inform.pucp.edu.pe/~inf232/Ntfs/ntfs_doc_v0.5/concepts/data_runs.html
						// In some exceptional cases $REPAIR file, a data offset is specified, however, the datarun is empty as repair might not be configured
						// To deal with this, check if ofssetToAttributeData doesn't overflow the attribute array, in case it does, lets ignore this.
					
						if (int(ofssetToAttributeData) + 1) >= len(attribute){
							fmt.Println("Exceptional case where $DATA is empty, ignoring this entry")
						}	
					
						if (int(ofssetToAttributeData) + 1) < len(attribute){
							var dataRun internal.DATA_RUN
							binary.Read(bytes.NewBuffer(attribute[ofssetToAttributeData:ofssetToAttributeData+1]), binary.LittleEndian, &dataRun.Nimble)
							// Weird exception case, where there is no data at all or we encounter a sparce / compressed data run
							// A nimble should have 2 values and shoulnd't be 0, hence comparing if larger than 9
							if(dataRun.Nimble >= 9){
								dataRun.ClusterCountLength, dataRun.ClusterOffsetLength = internal.ParseNimble(dataRun.Nimble)
	
								tmpStartOffset := ofssetToAttributeData+1+ uint16(dataRun.ClusterCountLength)
								tmpStopOffset := ofssetToAttributeData+1+uint16(dataRun.ClusterCountLength)+uint16(dataRun.ClusterOffsetLength)
								fileInformation.FullDataOffset =  uint64(NTFSOffset) + (uint64(clusterSize) * uint64(getNumber(attribute[tmpStartOffset:tmpStopOffset])))
							}
						}
					}				
				}
				// Updating attribute offset and making sure we can iterateMFT
				offsetToAttribute = offsetToAttribute + attributeLength				
			}
			binary.Read(bytes.NewBuffer(recordBuffer[offsetToAttribute:offsetToAttribute +4]), binary.LittleEndian, &checkEndMarker)
		}
	}
	return fileInformation
}

func ParseNTFSHeader(driveLocation string, NTFSHeaderOffset uint32, NTFSHeaderSize uint32) internal.NTFS_BOOT_PARTITION{
	var ntfsHeader internal.NTFS_BOOT_PARTITION
	handle, error := os.Open(driveLocation)
	if(error == nil){
		ntfsHeaderBuffer := make([]byte, NTFSHeaderSize)
		handle.Seek(int64(NTFSHeaderOffset),0)		
		handle.Read(ntfsHeaderBuffer)
		// Parsing start of NTFS_block
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[0:3]), binary.LittleEndian, &ntfsHeader.Jumpinstruction)
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[3:11]), binary.LittleEndian, &ntfsHeader.OemID)
		// Parsing Bios Parameter Block (25 bytes)
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[11:13]), binary.LittleEndian, &ntfsHeader.BytesPerSector)
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[13:14]), binary.LittleEndian, &ntfsHeader.SectorPerCluster)
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[14:16]), binary.LittleEndian, &ntfsHeader.ReservedSectors)
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[16:19]), binary.LittleEndian, &ntfsHeader.AlwaysZero)
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[19:21]), binary.LittleEndian, &ntfsHeader.Unused)
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[21:22]), binary.LittleEndian, &ntfsHeader.MediaDescription)
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[22:24]), binary.LittleEndian, &ntfsHeader.AlsoAlwaysZero)
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[24:26]), binary.LittleEndian, &ntfsHeader.SectorsPerTrack)
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[26:28]), binary.LittleEndian, &ntfsHeader.NumberOfHeads)
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[28:32]), binary.LittleEndian, &ntfsHeader.HiddenSectors)
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[32:36]), binary.LittleEndian, &ntfsHeader.UnusedTwo)
		// Parsing Extended Bios Parameter Block (48 bytes)
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[36:40]), binary.LittleEndian, &ntfsHeader.UnusedThree)
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[40:48]), binary.LittleEndian, &ntfsHeader.TotalSectors)
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[48:56]), binary.LittleEndian, &ntfsHeader.MFTOffset)
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[56:64]), binary.LittleEndian, &ntfsHeader.MFTMirrorOffset)
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[64:68]), binary.LittleEndian, &ntfsHeader.ClusterPerFileRecord)
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[68:72]), binary.LittleEndian, &ntfsHeader.ClusterPerIndexBlock)
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[72:80]), binary.LittleEndian, &ntfsHeader.VolumeSerialNumber)
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[80:84]), binary.LittleEndian, &ntfsHeader.Checksum)
		// Parsing start of NTFS_block
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[84:510]), binary.LittleEndian, &ntfsHeader.BootstrapCode)
		binary.Read(bytes.NewBuffer(ntfsHeaderBuffer[510:512]), binary.LittleEndian, &ntfsHeader.EndOfSectionMarker)
		// done parsing :)
	}
	handle.Close()
	return ntfsHeader
}

func ParseMFTEntry(mftRecordBuffer []byte) internal.MFT_ENTRY{
	var mftEntry internal.MFT_ENTRY
	
	binary.Read(bytes.NewBuffer(mftRecordBuffer[0:4]), binary.LittleEndian, &mftEntry.MagicNumber)
	binary.Read(bytes.NewBuffer(mftRecordBuffer[4:6]), binary.LittleEndian, &mftEntry.OffsetToUpdate)
	binary.Read(bytes.NewBuffer(mftRecordBuffer[6:8]), binary.LittleEndian, &mftEntry.SizeInWordsOfUpdateSequence)
	binary.Read(bytes.NewBuffer(mftRecordBuffer[8:16]), binary.LittleEndian, &mftEntry.LogFileSequenceNumber)
	binary.Read(bytes.NewBuffer(mftRecordBuffer[16:18]), binary.LittleEndian, &mftEntry.SequenceNumber)
	binary.Read(bytes.NewBuffer(mftRecordBuffer[18:20]), binary.LittleEndian, &mftEntry.HardLinkCounter)
	binary.Read(bytes.NewBuffer(mftRecordBuffer[20:22]), binary.LittleEndian, &mftEntry.OffsetToFirstAttribute)
	binary.Read(bytes.NewBuffer(mftRecordBuffer[22:24]), binary.LittleEndian, &mftEntry.Flags)
	binary.Read(bytes.NewBuffer(mftRecordBuffer[24:28]), binary.LittleEndian, &mftEntry.SizeRecord)
	binary.Read(bytes.NewBuffer(mftRecordBuffer[28:32]), binary.LittleEndian, &mftEntry.AllocatedSizeRecord)
	binary.Read(bytes.NewBuffer(mftRecordBuffer[32:40]), binary.LittleEndian, &mftEntry.FileReference)
	binary.Read(bytes.NewBuffer(mftRecordBuffer[40:42]), binary.LittleEndian, &mftEntry.NextAttributeID)
	binary.Read(bytes.NewBuffer(mftRecordBuffer[42:44]), binary.LittleEndian, &mftEntry.XPONLY_boundary)
	binary.Read(bytes.NewBuffer(mftRecordBuffer[44:48]), binary.LittleEndian, &mftEntry.XPONLY_RecordNumber)
	
	return mftEntry
}

func parsePartition(partitionBuffer []byte)internal.PARTITIONENTRY{
	var partitionEntry internal.PARTITIONENTRY
	
	binary.Read(bytes.NewBuffer(partitionBuffer[0:16]), binary.LittleEndian, &partitionEntry.PartitionGUID)
	binary.Read(bytes.NewBuffer(partitionBuffer[16:32]), binary.LittleEndian, &partitionEntry.UniquePartitionGUID)
	binary.Read(bytes.NewBuffer(partitionBuffer[32:40]), binary.LittleEndian, &partitionEntry.StartingLBA)
	binary.Read(bytes.NewBuffer(partitionBuffer[40:48]), binary.LittleEndian, &partitionEntry.EndingLBA)
	binary.Read(bytes.NewBuffer(partitionBuffer[48:56]), binary.LittleEndian, &partitionEntry.Attributes)
	binary.Read(bytes.NewBuffer(partitionBuffer[56:128]), binary.LittleEndian, &partitionEntry.PartitionName)
	
	return partitionEntry
}


func ParsePartitions(driveLocation string, partitionTableOffset uint32, partitionTableEntrySize uint32, partitionTableEntries uint32) (int, []internal.PARTITIONENTRY){
	partitionsFound := 0
	var partitionArray []internal.PARTITIONENTRY
	handle, error := os.Open(driveLocation)
	if(error == nil){
		partitionTableBuffer := make([]byte, partitionTableEntries * partitionTableEntrySize)
		handle.Seek(int64(partitionTableOffset),0)		
		handle.Read(partitionTableBuffer)
		for i := uint32(0); i < partitionTableEntries; i++ {
			partitionEntry := partitionTableBuffer[i*partitionTableEntrySize:(i+1)*partitionTableEntrySize]
			if!(internal.IsEmptyBuffer(partitionEntry)){
				partitionArray = append(partitionArray, parsePartition(partitionEntry))
				partitionsFound++
			}
		}
	}
	handle.Close()
	return partitionsFound, partitionArray
}

// *** Parsers
func ParseGPTHeader(driveLocation string, logicalBlockAddressSize int64, LBAOffset int64) internal.GPTHEADER{
	var gptheader internal.GPTHEADER
	handle, error := os.Open(driveLocation)
	if(error == nil){
		gptBuffer := make([]byte, logicalBlockAddressSize)
		handle.Seek(logicalBlockAddressSize*LBAOffset,0)		
		handle.Read(gptBuffer)
		
		// Parse the buffer to GPT header
		binary.Read(bytes.NewBuffer(gptBuffer[0:8]), binary.LittleEndian, &gptheader.Signature)
		binary.Read(bytes.NewBuffer(gptBuffer[8:12]), binary.LittleEndian, &gptheader.Revision)
		binary.Read(bytes.NewBuffer(gptBuffer[12:16]), binary.LittleEndian, &gptheader.HeaderSize)
		binary.Read(bytes.NewBuffer(gptBuffer[16:20]), binary.LittleEndian, &gptheader.Crc32)
		binary.Read(bytes.NewBuffer(gptBuffer[20:24]), binary.LittleEndian, &gptheader.Reserved)
		binary.Read(bytes.NewBuffer(gptBuffer[24:32]), binary.LittleEndian, &gptheader.CurrentLBA)
		binary.Read(bytes.NewBuffer(gptBuffer[32:40]), binary.LittleEndian, &gptheader.BackupLBA)
		binary.Read(bytes.NewBuffer(gptBuffer[40:48]), binary.LittleEndian, &gptheader.FirstLBA)
		binary.Read(bytes.NewBuffer(gptBuffer[48:56]), binary.LittleEndian, &gptheader.LastLBA)
		binary.Read(bytes.NewBuffer(gptBuffer[56:72]), binary.LittleEndian, &gptheader.DiskGUID)
		binary.Read(bytes.NewBuffer(gptBuffer[72:80]), binary.LittleEndian, &gptheader.PartitionEntriesLBA)
		binary.Read(bytes.NewBuffer(gptBuffer[80:84]), binary.LittleEndian, &gptheader.NumberOfPartitions)
		binary.Read(bytes.NewBuffer(gptBuffer[84:88]), binary.LittleEndian, &gptheader.PartitionEntrySize)
		binary.Read(bytes.NewBuffer(gptBuffer[88:92]), binary.LittleEndian, &gptheader.Crc32PartitionEntry)
	}
	handle.Close()
	return gptheader
}


func GetMFTOffsetLocationsFromMFT(driveLocation string, MFTOffset uint32, recordSize int64, NTFSOffset uint32)[]int{
	// This function parses the $DATA entry of the $MFT file, to find all MFT blocks and zones
	// No need to parse the full record, this will be done through a more systematic iterator.
	// Flag indicating $DATA attribute = 0x80 https://learn.microsoft.com/en-us/windows/win32/devnotes/attribute-list-entry,
	handle, error := os.Open(driveLocation)
	dataType := uint16(128)	// Searching for attribute of type 0x80, e.g. 128 in dec
	var mftClusterOffsets []int

	if(error == nil){
		mftRecordBuffer := make([]byte, recordSize)
		handle.Seek(int64(MFTOffset),0)		
		handle.Read(mftRecordBuffer)

		var offsetToAttribute uint16
		var attributeType uint16
		var attributeLength uint16
		
		binary.Read(bytes.NewBuffer(mftRecordBuffer[20:22]), binary.LittleEndian, &offsetToAttribute)
		binary.Read(bytes.NewBuffer(mftRecordBuffer[offsetToAttribute:offsetToAttribute+4]), binary.LittleEndian, &attributeType)
		binary.Read(bytes.NewBuffer(mftRecordBuffer[offsetToAttribute+4:offsetToAttribute+4+4]), binary.LittleEndian, &attributeLength)
		
		// This only works because we are certain that $MFT actually has the $DATA attribute
		for(attributeType != dataType){
			offsetToAttribute = offsetToAttribute + attributeLength
			binary.Read(bytes.NewBuffer(mftRecordBuffer[offsetToAttribute:offsetToAttribute+4]), binary.LittleEndian, &attributeType)
			binary.Read(bytes.NewBuffer(mftRecordBuffer[offsetToAttribute+4:offsetToAttribute+4+4]), binary.LittleEndian, &attributeLength)
		}
		fmt.Printf("  --> $DATA attribute of $MFT found at record offset: %d\n", offsetToAttribute)
		// The name of the Attribute is $DATA, the offset is stored in the header on the 10th to twelfth byte. Hence the data runs start at attributes offset + name offset
		
		var offsetToDATAAttribute uint16
		var dataAttributeLenght uint16
		var dataRunOffset uint16
		fullDiskMFTBlockOffset := int64(NTFSOffset)
		binary.Read(bytes.NewBuffer(mftRecordBuffer[offsetToAttribute+4:offsetToAttribute+8]), binary.LittleEndian, &dataAttributeLenght)
		binary.Read(bytes.NewBuffer(mftRecordBuffer[offsetToAttribute+10:offsetToAttribute+12]), binary.LittleEndian, &offsetToDATAAttribute)
		dataRunCounter := int(dataAttributeLenght) - int(offsetToDATAAttribute)
		dataRunOffset = offsetToAttribute + offsetToDATAAttribute

		// Post condition for looping through the data runs
		//because attribute types are assending, and the first byte of the attribute header contains a type, we can assume that we are past our data runs if the nimble exceeds 0x44 (68) or is just 0
		for (dataRunCounter > 3){
			var dataRun internal.DATA_RUN
			// To save space, the dataRun varies in space, the only thing we know for sure is that the length is the first byte: http://inform.pucp.edu.pe/~inf232/Ntfs/ntfs_doc_v0.5/concepts/data_runs.html
			binary.Read(bytes.NewBuffer(mftRecordBuffer[dataRunOffset:dataRunOffset+1]), binary.LittleEndian, &dataRun.Nimble)
			dataRun.ClusterCountLength, dataRun.ClusterOffsetLength = internal.ParseNimble(dataRun.Nimble)
			clusterStart := int(dataRunOffset) + 1 + dataRun.ClusterCountLength

			// **************** Welcome to casting, converting and formatting hell! **************** 
			offsetToMFTBlock := internal.ConvertClusterOffsetHexFromDatarunToDecimalOffset(dataRun.ClusterOffsetLength, mftRecordBuffer, clusterStart)
			
			// Fix the offsets to the next datarun
			dataRunOffset = dataRunOffset + 1 + uint16(dataRun.ClusterCountLength) + uint16(dataRun.ClusterOffsetLength)	//need + 1 as this indicates the length byte
			dataRunCounter = int(dataRunCounter) - 1 - dataRun.ClusterCountLength - dataRun.ClusterOffsetLength
			
			//To get to the right offset, we need to add the offset from the previous data run 
			fullDiskMFTBlockOffset = int64(fullDiskMFTBlockOffset) + int64(offsetToMFTBlock)
			mftClusterOffsets = append(mftClusterOffsets, int(fullDiskMFTBlockOffset))
		}

	}
	handle.Close()
	return mftClusterOffsets
}