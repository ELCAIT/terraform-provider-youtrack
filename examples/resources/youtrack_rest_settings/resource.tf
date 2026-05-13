resource "youtrack_rest_settings" "example" {
  allow_all_origins = false
  allowed_origins = [
    "https://example.com",
    "https://api.example.com"
  ]
}
