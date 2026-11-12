pub trait Named { // def rs.Named
    fn name(&self) -> String; // def rs.Named.name
}

pub trait Shape: Named + Clone + Debug { // super Named -> rs.Named; nosuper Clone; nosuper Debug
    fn area(&self) -> f64; // def rs.Shape.area

    fn describe(&self) -> String { // def rs.Shape.describe
        self.name() // call name -> rs.Named.name
    }
}

// Shares the method name with Named and is not a supertrait of Shape.
pub struct Plain;

impl Plain {
    pub fn name(&self) -> String { // def rs.Plain.name
        String::new()
    }
}

// A struct that shares its name with a standard trait. It is not what a trait
// bound `Debug` means.
pub struct Debug; // def rs.Debug
