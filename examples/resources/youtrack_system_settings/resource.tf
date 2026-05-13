resource "youtrack_system_settings" "server_config" {
  administrator_email         = "test@youtrack.com"
  max_export_items            = 500
  max_upload_file_size        = 10485760
  allow_statistics_collection = false
  is_application_read_only    = false
  base_url                    = "localhost:8080"
}
