package toolchain

import "testing"

func TestCandidateRegistryHasUniqueLanguagesAndTools(t *testing.T){
	seen:=map[string]bool{}
	for _,candidate:=range Candidates(){
		if candidate.Language==""{t.Fatal("candidate has empty language")}
		if seen[candidate.Language]{t.Fatalf("duplicate language %q",candidate.Language)}
		seen[candidate.Language]=true
		if len(candidate.Tools)==0{t.Fatalf("%s has no tool candidates",candidate.Language)}
	}
	for _,required:=range []string{"Go","Python","TypeScript","Rust","Java","C","C++","C#","Kotlin","MQL5"}{
		if !seen[required]{t.Fatalf("required language %q missing from registry",required)}
	}
}
