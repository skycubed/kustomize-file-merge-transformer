package main

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"os"
	"sigs.k8s.io/kustomize/api/hasher"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/api/types"
	"strings"
)

const NewLine = "\n"

func exitOnErr(err error) {
	if err != nil {
		os.Exit(1)
	}
}

type FileMergeTransformer struct {
	types.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
}

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

var targets multiFlag

type envVar struct {
	Var             string
	ReplacementName string
}
type targetArg struct {
	TargetFile string
	envVars    []envVar
}

func parseTargetArgs(args []string) []*targetArg {

	trgArgs := make([]*targetArg, len(args))

	for i, arg := range args {
		trgArgs[i] = parseTargetArg(arg)
	}

	return trgArgs
}

func parseTargetArg(arg string) *targetArg {

	parts := strings.Split(arg, ",")

	trgArg := &targetArg{
		TargetFile: parts[0],
		envVars:    make([]envVar, 0),
	}

	if len(parts) > 1 {
		for _, part := range parts[1:] {
			pairs := strings.Split(part, "=")
			v := envVar{
				Var: pairs[0],
			}
			if len(pairs) > 1 {
				v.ReplacementName = pairs[1]
			}
			trgArg.envVars = append(trgArg.envVars, v)
		}
	}

	return trgArg
}

type FilePrefix = string

func main() {

	if len(os.Args) < 3 {
		os.Exit(1)
	}

	fSet := flag.NewFlagSet("", flag.ExitOnError)
	fSet.Var(&targets, "target", "the target to merge (can be called multiple times)")
	err := fSet.Parse(os.Args[2:])
	exitOnErr(err)

	trgArgs := parseTargetArgs(targets)

	data, err := os.ReadFile(os.Args[1])
	exitOnErr(err)

	var t FileMergeTransformer
	err = yaml.Unmarshal(data, &t)
	exitOnErr(err)

	resData, err := io.ReadAll(os.Stdin)
	exitOnErr(err)

	f := resmap.NewFactory(resource.NewFactory(&hasher.Hasher{}))

	rm, err := f.NewResMapFromBytes(resData)
	exitOnErr(err)

	rmClone := rm.DeepCopy()

	for _, r := range rm.Resources() {

		if r.GetKind() == "ConfigMap" && r.GetName() == t.Name {

			clone := r.DeepCopy()

			toMerge := make(map[string]FilePrefix, 0)

			dataMap := r.GetDataMap()

			newDataMap := clone.GetDataMap()

			for _, trgArg := range trgArgs {

				fileName := trgArg.TargetFile

				if _, ok := dataMap[fileName]; ok {

					idx := strings.LastIndex(fileName, ".")
					if idx == -1 {
						os.Exit(1) // file extension required
					}

					toMerge[fileName] = fileName[:idx]

					for _, eVar := range trgArg.envVars {

						varName := eVar.Var

						if varContent, ok := dataMap[varName]; ok {

							propName := eVar.Var
							if eVar.ReplacementName != "" {
								propName = eVar.ReplacementName
							}

							toAdd := fmt.Sprintf("%s=%s", propName, varContent)

							if _, ok := newDataMap[fileName]; !ok {
								newDataMap[fileName] = toAdd
								continue
							}

							existing := newDataMap[fileName]

							if existing[len(existing)-1:] != NewLine {
								toAdd = NewLine + toAdd
							}

							newDataMap[fileName] = newDataMap[fileName] + toAdd

							delete(newDataMap, varName)

						}

					}

					continue

				}

			}

			for targetFileName, prefix := range toMerge {

				for fileName, content := range dataMap {

					cLen := len(content)

					if cLen == 0 {
						os.Exit(1)
					}

					if fileName == targetFileName || !strings.HasPrefix(fileName, prefix) {
						continue
					}

					toAdd := content

					if content[cLen-1:] != NewLine {
						toAdd = NewLine + toAdd
					}

					newDataMap[targetFileName] = newDataMap[targetFileName] + toAdd

					delete(newDataMap, fileName)

				}

			}

			clone.SetDataMap(newDataMap)

			_, err = rmClone.Replace(clone)
			exitOnErr(err)

		}

	}

	resData, err = rmClone.AsYaml()
	exitOnErr(err)

	_, err = fmt.Fprint(os.Stdout, string(resData))
	exitOnErr(err)

}
