# Master File Table 2 SQLite (MFT2SQL)

**MFS2SQL** is go-based parser for the Windows Master File Table (MFT). It allows you to query,analyse, and carve out (protected/hidden) files through forensic-grade access. MFS2SQL works by directly scanning the physical disk using — granting low-level access to raw bytes across all sectors. It begins by parsing the GPT (GUID Partition Table) to determine how many partitions exist and identify the one formatted with NTFS. Once the NTFS partition is located, it reads its header to calculate key offsets and locate the Master File Table (MFT), which contains the record-based index of all files on the volume — including deleted and hidden ones.

The MFT itself is a structured, block-based system that functions like a paged metadata database. Each record represents a unique file or folder and includes rich metadata: filename(s), parent relationships, timestamps, flags, disk location (offset and length), and even security descriptors. While the layout might resemble a linked list, NTFS uses internal mappings and attribute indirection to stitch together fragmented or extended records. That complexity allows for robust recovery and forensic inspection, especially since deleted file records can linger in the MFT long after removal. By reconstructing these records, MFS2SQL translates disk-level artifacts into searchable SQL entries.

> 🔒 **Administrator privileges are required** for accessing low-level disk interfaces such as `\\.\physicaldrive0`.

---

## ⚙️ Setup & First Run

Before querying or carving, initialize the database with:

```bash
MFT2SQL.exe -dumpMode 2
```
This indexes all MFT entries and recursively resolves their full paths. ⏱️ It takes ~16 minutes on a full disk scan (SSD speeds may vary).

---

## 💻 Features & Highlights

- 🔍 Converts raw MFT records into structured SQL records
- 📂 Automatically reconstructs full file paths via parent-child relationships
- 📎 Tracks file size, disk offset, activity status, and folder flags
- 🧬 Supports direct file carving using metadata from MFT
- 🗃️ Enables SQL-indexed lookup for flexibility

| **Flag**           | **Description**                                                            |
|--------------------|----------------------------------------------------------------------------|
| `-dbFile string`   | SQLite DB name (default `"MFTDB.db"`). Applies to -dumpFile and -getFileLocation |
| `-carve`           | Carve a file from disk. Requires `-fileOffset` and `-fileLength`.         |
| `-fileLength int`  | Length of the file to carve (in bytes).                                   |
| `-fileOffset int`  | Disk offset to start carving from (in bytes).                             |
| `-dumpFile string` | Dump MFT to a custom database or file output. Options: `1=screen`, `2=SQL`. |
| `-deviceLocation string`  | Disk source to scan (default `"\\\\.\\physicaldrive0"`).                  |
| `-dumpMode int`    | MFT dump output: `1=screen`, `2=SQL`.                                     |
| `-getFileLocation string` | Lookup file offset and length by full NTFS path.                          |
| `-help`            | Show help and usage banner.                                                |

---

## 🧉 Buy Me a Coffee

If this tool saved you hours of scripting or made your investigation easier, or got you valuable credentials during a penetration test — feel free to buy me a coffee:

👉 [https://buymeacoffee.com/jeroens](https://buymeacoffee.com/jeroens)

Much appreciated! 🧠☕

---

## 🧪 Example Commands

**Dump MFT to a custom database output file:**
```bash
go run MFT2SQL.go -dbFile custom.db -dumpMode 2
```

**Fetch location data of a file:**
```bash
go run MFT2SQL.go -dbFile custom.db -getFileLocation Windows\System32\config\SAM
```

**Carve file (SAM file in this case) and store it in custom output:**
```bash
go run MFT2SQL.go -carve -fileOffset  28721337472  -fileLength  131004 -dumpFile SAMFile.txt
```

## 📜 License

This project is licensed under the [Apache License 2.0](https://raw.githubusercontent.com/MFT2SQL/MFT2SQL/refs/heads/main/LICENSE).  
See `LICENSE.md` for the full license text and terms of use.


