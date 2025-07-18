package internal

// *** Structs to capture data in a structurized way
type GPTHEADER struct{
	 // Based on table from: http://ntfs.com/guid-part-table.htm
	Signature [8]byte 
	Revision [4]byte
	HeaderSize uint16		//92 should be used, leaving 420 zeros at the end
	Crc32 [4]byte
	Reserved [4]byte
	CurrentLBA uint32
	BackupLBA uint32
	FirstLBA uint32
	LastLBA uint32
	DiskGUID [16]byte
	PartitionEntriesLBA uint32
	NumberOfPartitions uint32
	PartitionEntrySize uint32
	Crc32PartitionEntry [4]byte
	EmptySpace [0]byte
}

type PARTITIONENTRY struct{
	// Based on table from: https://wiki.osdev.org/GPT
	PartitionGUID [16]byte
	UniquePartitionGUID [16]byte
	StartingLBA uint32
	EndingLBA uint32
	Attributes [8]byte
	PartitionName [72]byte
}

type NTFS_BOOT_PARTITION struct{
	// Based on informatiom from: https://learn.microsoft.com/en-us/previous-versions/windows/it-pro/windows-2000-server/cc976796(v=technet.10)?redirectedfrom=MSDN
	Jumpinstruction [3]byte
	OemID [8]byte
	BytesPerSector uint16			//Start of Bios Parameter Block (BPB)
	SectorPerCluster uint8
	ReservedSectors [2]byte
	AlwaysZero [3]byte
	Unused [2]byte
	MediaDescription [1]byte
	AlsoAlwaysZero [2]byte
	SectorsPerTrack [2]byte
	NumberOfHeads [2]byte
	HiddenSectors [4]byte
	UnusedTwo [4]byte
	UnusedThree [4]byte				//Start of Extended BPB
	TotalSectors [8]byte
	MFTOffset uint32
	MFTMirrorOffset [8]byte
	ClusterPerFileRecord [4]byte
	ClusterPerIndexBlock [4]byte
	VolumeSerialNumber [8]byte
	Checksum [4]byte				// End EBPB
	BootstrapCode [426]byte
	EndOfSectionMarker [2]byte
}

type MFT_ENTRY struct{
	// Based on information from: https://flatcap.github.io/linux-ntfs/ntfs/concepts/file_record.html
	MagicNumber [4]byte
	OffsetToUpdate [2]byte
	SizeInWordsOfUpdateSequence [2]byte
	LogFileSequenceNumber [8]byte
	SequenceNumber [2]byte
	HardLinkCounter [2]byte
	OffsetToFirstAttribute [2]byte
	Flags [2]byte
	SizeRecord [4]byte
	AllocatedSizeRecord [4]byte
	FileReference [8]byte
	NextAttributeID [2]byte
	XPONLY_boundary [2]byte
	XPONLY_RecordNumber [4]byte
	Attributes[]byte
}

type MFT_ENTRY_ATTRIBUTE struct{
	// Based on information from: http://inform.pucp.edu.pe/~inf232/Ntfs/ntfs_doc_v0.5/concepts/attribute_header.html
	// Default one: non-resident, no-name (NRNN)
	AttributeType [4]byte
	Length [4]byte
	NonResidentFlag [1]byte
	NameLenght [1]byte
	NameOffset [2]byte
	Flags [2]byte
	AttributeID [2]byte
	AttributeLenght [4]byte
	OffsetToAttribute [2]byte
	IndexFlag [1]byte
	Padding [1]byte
	AttributeName []byte	
}

type DATA_RUN struct{
	Nimble uint8
	ClusterCountLength int
	ClusterOffsetLength int
	ClusterCount int64
	AbsoluteOffsetWithinNTFSPartition int64
}

type FILE_INFO struct{
	RecordID uint32
	IsFolder bool
	IsActive bool
	FileName string
	FileCreatedUTCWinFileEpoch uint32
	FileModifiedUTCWinFileEpoch uint32
	FileRecordModifiedUTCWinFileEpoch uint32
	FileLastReadUTCWinFileEpoch uint32
	FilePermissionFlag uint32
	FileOwnerID uint16
	ParentDirectory uint32
	DataLength uint32
	FullDataOffset uint64	//This should include the NTFS offset as well!
}