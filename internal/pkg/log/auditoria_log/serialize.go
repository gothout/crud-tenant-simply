package auditoria_log

import (
	"encoding/json"
	"fmt"
)

// SerializeData tenta converter o payload para JSON; se falhar, retorna a
// representação formatada com fmt.
func SerializeData(data interface{}) string {
	if data == nil {
		return ""
	}

	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf("%+v", data)
	}

	return string(raw)
}
