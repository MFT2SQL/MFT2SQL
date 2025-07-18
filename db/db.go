package db

import "fmt"
import "database/sql"
import _ "modernc.org/sqlite"			

 
var Database *sql.DB
 
// Structure to support sql batch insert, to increase application performance
var (
    Tx        *sql.Tx
    Stmt      *sql.Stmt
    BatchSize = 10000
    Batch     = 0
    InsertCounter = 0
)

// Structure to support reconstructing full paths
type sqlDBFileEntry struct {
    FID       int
    RID       int
    ParentID  int
    Filename  string
    FullPath  string
}

/* Database functionality */

func SetUpSQLiteDB(dbFile string) bool {
    var err error
    Database, err = sql.Open("sqlite", dbFile) // Use "sqlite" if you’re using modernc.org/sqlite
    if err != nil {
        fmt.Println("[!] Error opening database:", err)
        return false
    }

    // Ensure the DB connection is alive
    if err = Database.Ping(); err != nil {
        fmt.Println("[!] Failed to connect to database:", err)
        return false
    }

    // Clear previous data by dropping the table, if it exists
    _, err = Database.Exec(`DROP TABLE IF EXISTS files`)
    if err != nil {
        fmt.Println("[!] Error dropping table:", err)
        return false
    }

    // Recreate the table
    _, err = Database.Exec(`
        CREATE TABLE files (
            FID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, RID INTEGER, parentID INTEGER, filename TEXT, fileOffset INTEGER, fileLength INTEGER, isFolder INTEGER, isActive INTEGER, fullPath TEXT
        )
    `)
    if err != nil {
        fmt.Println("[!] Error creating table:", err)
        return false
    }
	
	// Create indexes to accelerate recursive path queries
    _, err = Database.Exec(`CREATE INDEX idx_rid ON files(RID)`)
    if err != nil {
        fmt.Println("[!] Error creating RID index:", err)
        return false
    }

    _, err = Database.Exec(`CREATE INDEX idx_parent ON files(parentID)`)
    if err != nil {
        fmt.Println("[!] Error creating parentID index:", err)
        return false
    }

    fmt.Println("[+] Database is clean and ready to use")
	return true
}


/* Dump to DB functionality */

func FlushBatch() {
    if Tx == nil {
        return
    }
    err := Stmt.Close()
    if err != nil {
        fmt.Println("[!] Error closing statement:", err)
    }
    err = Tx.Commit()
    if err != nil {
        fmt.Println("[!] Error committing transaction:", err)
    } else {
        InsertCounter += Batch
        fmt.Printf("[.] Committed batch of %d records. Total inserted: %d\n", Batch, InsertCounter)
    }
    // Reset for next batch
    Tx = nil
    Stmt = nil
    Batch = 0
}


func InsertFileRecord(RID int, filename string, parentID int, isFolder int, isActive int, fullOffset int, dataLength int) {
    if Tx == nil {
        var err error
        Tx, err = Database.Begin()
        if err != nil {
            fmt.Println("[!] Failed to begin transaction:", err)
            return
        }
        Stmt, err = Tx.Prepare("INSERT INTO files (RID, parentID, filename, fileOffset, fileLength, isFolder, isActive) VALUES (?, ?, ?, ?, ?, ?, ?)")
        if err != nil {
            fmt.Println("[!] Failed to prepare statement:", err)
            return
        }
    }

    _, err := Stmt.Exec(RID, parentID, filename, fullOffset, dataLength, isFolder, isActive)
    if err != nil {
        fmt.Println("[!] Insert error:", err)
        return
    }

    Batch++
    if Batch%BatchSize == 0 {
        FlushBatch()
    }
}


/* enrichment of collected data */
func fetchAllFiles() (map[int]*sqlDBFileEntry, error) {
    rows, err := Database.Query("SELECT FID, RID, parentID, filename FROM files")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    entries := make(map[int]*sqlDBFileEntry)
    for rows.Next() {
        var f sqlDBFileEntry
        if err := rows.Scan(&f.FID, &f.RID, &f.ParentID, &f.Filename); err != nil {
            return nil, err
        }
        entries[f.RID] = &f
    }
    return entries, nil
}

func buildFullPath(entries map[int]*sqlDBFileEntry, rid int) string {
    entry, ok := entries[rid]
    if !ok {
        return ""
    }
    if entry.FullPath != "" {
        return entry.FullPath
    }

    if entry.ParentID == entry.RID || entry.ParentID == 5 {
        entry.FullPath = entry.Filename
    } else {
        parentPath := buildFullPath(entries, entry.ParentID)
        if parentPath == "" {
            entry.FullPath = entry.Filename
        } else {
            entry.FullPath = parentPath + `\` + entry.Filename
        }
    }
    return entry.FullPath
}

func UpdateFullpaths() {
    fmt.Println("\n[+] Building full paths for all entries...")

    // Step 1: Fetch all entries into memory
    files, err := fetchAllFiles()
    if err != nil {
        fmt.Println("[!] Failed to load records:", err)
        return
    }

    // Step 2: Build fullpaths recursively
    for rid := range files {
        _ = buildFullPath(files, rid)
    }

    // Step 3: Begin bulk update
    Tx, err := Database.Begin()
    if err != nil {
        fmt.Println("[!] Failed to begin transaction:", err)
        return
    }

    Stmt, err := Tx.Prepare("UPDATE files SET fullpath = ? WHERE RID = ?")
    if err != nil {
        fmt.Println("[!] Failed to prepare update statement:", err)
        return
    }

    count := 0
    for rid, entry := range files {
        if entry.FullPath != "" {
            _, err := Stmt.Exec(entry.FullPath, rid)
            if err != nil {
                fmt.Printf("[!!] Failed to update RID %d: %v\n", rid, err)
            }
            count++
            if count%100000 == 0 {
                fmt.Printf("[+] Updated %d fullpaths...\n", count)
            }
        }
    }

    Stmt.Close()
    err = Tx.Commit()
    if err != nil {
        fmt.Println("[!] Commit failed:", err)
        return
    }

    fmt.Println("[+] Fullpaths updated for", count, "records.")
}
