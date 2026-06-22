# Seed a fixture user with a plaintext password.
resource "clerk_user" "seed" {
  email_addresses = ["seed.user@example.com"]
  password        = "ChangeMe-123!"
  first_name      = "Seed"
  last_name       = "User"
  public_metadata = jsonencode({ role = "tester" })
}

# Migration-style seeding with a pre-hashed password.
resource "clerk_user" "migrated" {
  email_addresses      = ["migrated.user@example.com"]
  first_name           = "Migrated"
  last_name            = "User"
  password_digest      = "$2a$10$EpRnTzVlqHNP0.fUbXUwSOyuiXe/QLSUG6xNekdHgTGmrpHEfIoxm"
  password_hasher      = "bcrypt"
  skip_password_checks = true
}
