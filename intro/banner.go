package intro

import "fmt"

// Fancy banner
func ShowBannerAndIntro() {
    banner := `
==========================================================
   __  __ _____ _____    ____     ____   ___  _     
   |  \/  |  ___|_   _|  |___ \   / ___| / _ \| |    
   | |\/| | |_    | |      __) |  \___ \| | | | |    
   | |  | |  _|   | |     / __/    ___) | |_| | |___ 
   |_|  |_|_|     |_|    |_____|  |____/ \__\_\_____|
   
==========================================================
`

    intro := `
🔍 The MSF2SQL tool creates an SQL database based on the Windows Master File Table.
It allows you to directly access files, including their length and location on the physical disk.
Before using the tool, make sure to set-up the database: MSF2SQL -dumpMode 2
Please note, that the tool requires administrator priviledges for accessing \\.\
`
    fmt.Println(banner)
    fmt.Println(intro)
}