pub fn classify(flag: bool) -> i32 {
    for flag in [true, false] {
        if flag {
            return 1;
        } else if flag {
            return 2;
        }
    }
    if flag { 3 } else { 4 }
}
