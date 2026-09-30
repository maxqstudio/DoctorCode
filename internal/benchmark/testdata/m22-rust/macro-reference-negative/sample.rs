macro_rules! retain_symbol {
    ($symbol:ident) => {
        const _: &str = stringify!($symbol);
    };
}

fn macro_referenced_helper() {}

retain_symbol!(macro_referenced_helper);
