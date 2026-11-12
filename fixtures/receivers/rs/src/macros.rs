// A macro invocation is not a call of the function of the same name. fd bound
// std's `format!` to a test function named `format`.
fn format(width: usize) -> usize { // def rs.format
    width
}

fn describe(width: usize) -> String { // def rs.describe
    let rows = vec![width, width]; // call vec -> external
    println!("{}", rows.len()); // call println -> external
    format!("{}", width) // call format -> external
}
