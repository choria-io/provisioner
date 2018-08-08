// +build ignore

package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/alecthomas/template"
)

const ftempl = `package provision

// auto generated {{.Now}}

import (
	"encoding/base64"
	"encoding/json"

	addl "github.com/choria-io/mcorpc-agent-provider/mcorpc/ddl/agent"
)

var provddl = ` + "`{{.ProvDDL}}`" + `
var rpcuddl = ` + "`{{.RPCUtilDDL}}`" + `

// DDL is the agent DDL
var DDL = make(map[string]*addl.DDL)

func init() {
	DDL["choria_provision"] = &addl.DDL{}
	DDL["rpcutil"] = &addl.DDL{}

	ddl, _ := base64.StdEncoding.DecodeString(provddl)
	json.Unmarshal(ddl, DDL["choria_provision"])

	ddl, _ = base64.StdEncoding.DecodeString(rpcuddl)
	json.Unmarshal(ddl, DDL["rpcutil"])
}
`

type dat struct {
	ProvDDL    string
	RPCUtilDDL string
}

func (d dat) Now() string {
	return fmt.Sprintf("%s", time.Now())
}

func main() {
	provj, err := ioutil.ReadFile("agent/choria_provision.json")
	if err != nil {
		panic(fmt.Sprintf("Could not read agents spec file agent/choria_provision.json: %s", err))
	}

	rpcutj, err := ioutil.ReadFile("agent/rpcutil.json")
	if err != nil {
		panic(fmt.Sprintf("Could not read agents spec file agent/rpcutil.json: %s", err))
	}

	templ := template.Must(template.New("ddl").Parse(ftempl))

	f, err := os.Create("agent/ddl.go")
	if err != nil {
		panic(fmt.Sprintf("cannot create file agent/ddl.go: %s", err))
	}
	defer f.Close()

	input := dat{
		ProvDDL:    base64.StdEncoding.EncodeToString(provj),
		RPCUtilDDL: base64.StdEncoding.EncodeToString(rpcutj),
	}

	err = templ.Execute(f, input)
	if err != nil {
		panic(fmt.Sprintf("executing template:", err))
	}
}
