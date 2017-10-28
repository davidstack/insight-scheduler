package main
import(
	"encoding/json"
	"fmt"
)
func StructToJson(object interface{}) string {
	bytesinfo, err := json.Marshal(object)
	if err != nil {
		fmt.Println("call json marshal failed:")
		return ""
	} else {
		return string(bytesinfo)
	}
}
