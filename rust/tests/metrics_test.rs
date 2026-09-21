use incident_commander::metrics::{format_g, new_metrics};

#[test]
fn counter_accumulates() {
    let mut m = new_metrics();
    m.inc("incident_count", 1.0);
    m.inc("incident_count", 2.0);
    assert_eq!(m.counter("incident_count"), 3.0);
}

#[test]
fn counter_missing_is_zero() {
    assert_eq!(new_metrics().counter("nope"), 0.0);
}

#[test]
fn expose_contains_counter_gauge_and_type_lines() {
    let mut m = new_metrics();
    m.inc("incident_count", 1.0);
    m.inc("incident_count", 2.0);
    m.set("db_connections", 42.0);
    let out = m.expose();
    assert!(out.contains("incident_count 3"));
    assert!(out.contains("db_connections 42"));
    assert!(out.contains("# TYPE incident_count counter"));
    assert!(out.contains("# TYPE db_connections gauge"));
}

#[test]
fn expose_is_sorted_by_name() {
    let mut m = new_metrics();
    m.inc("b_metric", 1.0);
    m.inc("a_metric", 1.0);
    let out = m.expose();
    assert!(out.find("a_metric").unwrap() < out.find("b_metric").unwrap());
}

#[test]
fn expose_exact_layout() {
    let mut m = new_metrics();
    m.inc("z_counter", 3.0);
    m.inc("a_counter", 1.0);
    m.set("b_gauge", 1.5);
    m.set("a_gauge", 0.2);
    let expected = "# TYPE a_counter counter\na_counter 1\n# TYPE z_counter counter\nz_counter 3\n# TYPE a_gauge gauge\na_gauge 0.2\n# TYPE b_gauge gauge\nb_gauge 1.5\n";
    assert_eq!(m.expose(), expected);
}

#[test]
fn expose_empty_is_blank() {
    assert_eq!(new_metrics().expose(), "");
}

#[test]
fn set_overwrites_gauge() {
    let mut m = new_metrics();
    m.set("queue_depth", 3.0);
    m.set("queue_depth", 1.0);
    assert!(m.expose().contains("queue_depth 1\n"));
}

#[test]
fn format_g_integers() {
    assert_eq!(format_g(0.0), "0");
    assert_eq!(format_g(1.0), "1");
    assert_eq!(format_g(3.0), "3");
    assert_eq!(format_g(42.0), "42");
    assert_eq!(format_g(1000.0), "1000");
}

#[test]
fn format_g_simple_decimals() {
    assert_eq!(format_g(0.2), "0.2");
    assert_eq!(format_g(1.5), "1.5");
    assert_eq!(format_g(0.0001), "0.0001");
    assert_eq!(format_g(0.25), "0.25");
}

#[test]
fn format_g_negative() {
    assert_eq!(format_g(-1.0), "-1");
    assert_eq!(format_g(-0.5), "-0.5");
}
