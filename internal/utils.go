package internal

import "golang.org/x/sys/windows"
import "fmt"
import "strconv"
import "strings"

// Usability improvement
func IsAdmin() bool {
    var sid *windows.SID
    // SECURITY_NT_AUTHORITY is {0,0,0,0,0,5}
    ntAuthority := windows.SECURITY_NT_AUTHORITY
    err := windows.AllocateAndInitializeSid(&ntAuthority, 2,
        windows.SECURITY_BUILTIN_DOMAIN_RID,
        windows.DOMAIN_ALIAS_RID_ADMINS,
        0, 0, 0, 0, 0, 0, &sid)
    if err != nil {
        return false
    }

    token := windows.Token(0)
    isMember, err := token.IsMember(sid)
    return err == nil && isMember
}

// General supporting
// *** Supporting functions
func BoolToInt(value bool) int{
	var returnValue = 0
	if(value){
		returnValue = 1
	}
	if(!value){
		returnValue = 0
	}
	return returnValue
}

func IsEmptyBuffer(s []byte) bool {
    for _, v := range s {
        if v != 0 {
            return false
        }
    }
    return true
}

// Magic :)

func CalculateHexComplement(input string) (string){
	returnString := ""
	oppositesMap := make(map[string]string)
	oppositesMap["0"] = "f"
	oppositesMap["1"] = "e"
	oppositesMap["2"] = "d"
	oppositesMap["3"] = "c"
	oppositesMap["4"] = "b"
	oppositesMap["5"] = "a"
	oppositesMap["6"] = "9"
	oppositesMap["7"] = "8"
	oppositesMap["8"] = "7"
	oppositesMap["9"] = "6"
	oppositesMap["a"] = "5"
	oppositesMap["b"] = "4"
	oppositesMap["c"] = "3"
	oppositesMap["d"] = "2"
	oppositesMap["e"] = "1"
	oppositesMap["f"] = "0"
	
	for counter, _ := range input{
		returnString = returnString + oppositesMap[string(input[counter])]
	}
	return returnString
}

func ConvertClusterOffsetHexFromDatarunToDecimalOffset(offsetLength int, buffer []byte, offsetToMFTBlockOffsetInHex int) (int64){
	var returnOffset int64
	var offsetAsString string
	
	if((offsetLength) > 3){
				// If the first byte starts with a zero, we need to remove that zero, and add it to the end of the byte
				offsetAsString = offsetAsString + fmt.Sprintf("%x",buffer[offsetToMFTBlockOffsetInHex+3:offsetToMFTBlockOffsetInHex+4][0])
	}
	offsetAsString = offsetAsString + fmt.Sprintf("%x",buffer[offsetToMFTBlockOffsetInHex+2:offsetToMFTBlockOffsetInHex+3])
	offsetAsString = offsetAsString + fmt.Sprintf("%x",buffer[offsetToMFTBlockOffsetInHex+1:offsetToMFTBlockOffsetInHex+2])
	offsetAsString = offsetAsString + fmt.Sprintf("%x",buffer[offsetToMFTBlockOffsetInHex:offsetToMFTBlockOffsetInHex+1])
	offsetAsString = offsetAsString + "000"
	returnOffset,_ = strconv.ParseInt(offsetAsString, 16, 64)
	
	// Checking if the hex value represents a signed integer: https://www.d.umn.edu/~gshute/asm/signed.xhtml / https://github.com/libyal/libfsntfs/blob/main/documentation/New%20Technologies%20File%20System%20(NTFS).asciidoc
	// 1. get MSB of first Hex value, 2. (if signed, > 7, calculate the complement and add 1, transpose to decimal and add minus symbol)
	if( offsetAsString[0] > 64){
		// Calculate offset Complement
		complementHex := CalculateHexComplement(offsetAsString)
		returnOffset,_ = strconv.ParseInt(complementHex, 16, 64)
		returnOffset = (-1)*(returnOffset+1)
	}
	return returnOffset
}

func ParseNimble(nimble uint8) (int, int) {
    hexStr := fmt.Sprintf("%02x", nimble) // zorg altijd voor 2 karakters
    clusterOffsetLength, _ := strconv.Atoi(string(hexStr[0]))
    clusterCountLength, _ := strconv.Atoi(string(hexStr[1]))
    return clusterCountLength, clusterOffsetLength
}


func ParseNimble_old(nimble uint8) (int, int){
	var clusterCountLength int
	var clusterOffsetLength int
	slidedHexStringNimble := strings.Split(strconv.FormatInt(int64(nimble), 16),"")
	clusterCountLength,_ = strconv.Atoi(slidedHexStringNimble[1])
	clusterOffsetLength,_ = strconv.Atoi(slidedHexStringNimble[0])
	return clusterCountLength, clusterOffsetLength
}
