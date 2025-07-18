# Master File Table 2 SQLite (MFT2SQL)

**MFS2SQL** is go-based parser for the Windows Master File Table (MFT). It allows you to query,analyse, and carve out (protected/hidden) files through forensic-grade access

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
- 📂 Reconstructs full file paths via parent-child relationships
- 📎 Tracks file size, disk offset, activity status, and folder flags
- 🧬 Supports direct file carving using metadata from MFT
- 🗃️ Enables SQL-indexed lookup for flexibility


---

## 🧉 Buy Me a Coffee

If this tool saved you hours of scripting or made your investigation easier, or got you valuable credentials during a penetration test — feel free to support me:

👉 [https://buymeacoffee.com/jeroens](https://buymeacoffee.com/jeroens)

Much appreciated! 🧠☕

---

## 🧪 Example Commands

** Dump MFT to a custom database output file: **
```bash
go run MFT2SQL.go -dbFile custom.db -dumpMode 2
```

** Fetch location data of a file: **
```bash
go run MFT2SQL.go -dbFile custom.db -getFileLocation Windows\System32\config\SAM
```

## 📜 License

This project is licensed under the [Apache License 2.0](https://raw.githubusercontent.com/MFT2SQL/MFT2SQL/refs/heads/main/LICENSE).  
See `LICENSE.md` for the full license text and terms of use.

** Carve file (SAM file in this case) and store it in custom output: **
go run MFT2SQL.go -carve -fileOffset  28721337472  -fileLength  131004 -dumpFile SAMFile.txt
