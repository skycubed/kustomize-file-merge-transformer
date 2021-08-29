package main

import (
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

type Merge struct {
	Prefix   string
	FileName string
	Content  string
}

func exitOnErr(err error) {
	if err != nil {
		os.Exit(1)
	}
}

type FileMergeTransformer struct {
	types.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
}

func main() {

	if len(os.Args) < 3 {
		os.Exit(1)
	}

	data, err := os.ReadFile(os.Args[1])
	exitOnErr(err)

	var t FileMergeTransformer
	err = yaml.Unmarshal(data, &t)
	exitOnErr(err)

	mergeArg := strings.Join(os.Args[2:], " ")

	merges := make([]string, 0)
	for _, item := range strings.Split(mergeArg, ",") {
		merges = append(merges, strings.TrimSpace(item))
	}

	resData, err := io.ReadAll(os.Stdin)
	exitOnErr(err)

	f := resmap.NewFactory(resource.NewFactory(&hasher.Hasher{}))

	rm, err := f.NewResMapFromBytes(resData)
	exitOnErr(err)

	rmClone := rm.DeepCopy()

	for _, r := range rm.Resources() {

		if r.GetKind() == "ConfigMap" && r.GetName() == t.Name {

			clone := r.DeepCopy()

			toMerge := make([]Merge, 0)

			dataMap := r.GetDataMap()
			for _, fileName := range merges {
				if content, ok := dataMap[fileName]; ok {
					idx := strings.LastIndex(fileName, ".")
					if idx == -1 {
						os.Exit(1) // file extension required
					}
					toMerge = append(toMerge, Merge{fileName[:idx], fileName, content})
				}
			}

			newDataMap := clone.GetDataMap()

			for _, merge := range toMerge {

				for fileName, content := range dataMap {

					cLen := len(content)

					if cLen == 0 {
						os.Exit(1)
					}

					if fileName == merge.FileName || !strings.HasPrefix(fileName, merge.Prefix) {
						continue
					}

					toAdd := content

					if content[cLen-1:] != NewLine {
						toAdd = NewLine + toAdd
					}

					newDataMap[merge.FileName] = newDataMap[merge.FileName] + toAdd

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
