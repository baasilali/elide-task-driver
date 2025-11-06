# Nomad agent configuration for testing with Elide driver

plugin_dir = "./build/plugins"

plugin "elide" {
  config {
    daemon_socket = "/tmp/elide-daemon.sock"
    
    session_config {
      context_pool_size  = 10
      enabled_languages  = ["python", "javascript", "typescript"]
      enabled_intrinsics = ["io", "env"]
      memory_limit_mb    = 512
      enable_ai          = false
    }
  }
}

