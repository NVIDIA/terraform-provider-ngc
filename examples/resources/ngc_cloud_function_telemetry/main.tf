resource "ngc_cloud_function_telemetry" "log_telemetry" {
  endpoint           = "https://mock-grafana.net/otlp"
  protocol           = "HTTP"
  telemetry_provider = "GRAFANA_CLOUD"
  types              = ["METRICS", "LOGS", "TRACES"]
  secret = {
    name  = "ngc-terraform-test-log-telemetry"
    value = "123"
  }
}
