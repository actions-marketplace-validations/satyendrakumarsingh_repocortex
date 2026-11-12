package parse

import "testing"

func TestRustMacroInvocationsAreNotCalls(t *testing.T) {
	res := parse(t, "a.rs", `macro_rules! print_error { () => {}; }
fn format() {}
fn main() {
    let v = vec![1];
    let s = format!("{}", 1);
    format();
    print_error!("x");
    std::println!("y");
    obj.format();
}
`)
	wantList(t, "references", refRows(res), []string{
		"reference.call format recv=/bare args=?",
		"reference.call format recv=obj/name args=?",
		"reference.call.macro format recv=/bare args=?",
		"reference.call.macro print_error recv=/bare args=?",
		"reference.call.macro println recv=std/scope args=?",
		"reference.call.macro vec recv=/bare args=?",
	})
}
