# Example Nomad job for Elide task driver
# This will work once the daemon API is available

job "hello-python" {
  datacenters = ["dc1"]
  type        = "batch"

  group "python" {
    count = 1

    task "hello" {
      driver = "elide"

      config {
        script   = "local/hello.py"
        language = "python"

        elide_opts {
          memory_limit = 128
          enable_ai    = false
        }
      }

      # Inline the script
      template {
        data = <<EOF
#!/usr/bin/env python3
print("Hello from Elide on Nomad!")
print("This is a polyglot runtime")
for i in range(5):
    print(f"Count: {i}")
EOF
        destination = "local/hello.py"
        perms       = "755"
      }

      resources {
        cpu    = 100
        memory = 128
      }
    }
  }
}

