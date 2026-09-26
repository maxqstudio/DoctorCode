package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanRecognizesLanguagesAndSkipsDependencyDirectories(t *testing.T){
	root:=t.TempDir()
	writeFile(t,filepath.Join(root,"main.go"))
	writeFile(t,filepath.Join(root,"worker.py"))
	writeFile(t,filepath.Join(root,"README.md"))
	writeFile(t,filepath.Join(root,"node_modules","ignored.js"))
	result,err:=Scan(root)
	if err!=nil{t.Fatal(err)}
	if result.Files!=3{t.Fatalf("Files=%d; want 3",result.Files)}
	if result.RecognizedFiles!=2{t.Fatalf("RecognizedFiles=%d; want 2",result.RecognizedFiles)}
	if len(result.Languages)!=2{t.Fatalf("Languages=%#v; want two",result.Languages)}
	if result.Languages[0].Name!="Go"||result.Languages[1].Name!="Python"{t.Fatalf("Languages=%#v; want Go then Python",result.Languages)}
}

func writeFile(t *testing.T,path string){
	t.Helper()
	if err:=os.MkdirAll(filepath.Dir(path),0o755);err!=nil{t.Fatal(err)}
	if err:=os.WriteFile(path,[]byte("x\n"),0o644);err!=nil{t.Fatal(err)}
}
