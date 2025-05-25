class McpGo < Formula
  desc "Go implementation of the Model Context Protocol (MCP)"
  homepage "https://github.com/fredcamaral/mcp-memory"
  url "https://github.com/fredcamaral/mcp-memory/archive/refs/tags/v1.0.0.tar.gz"
  sha256 "YOUR_SHA256_HERE"
  license "MIT"
  head "https://github.com/fredcamaral/mcp-memory.git", branch: "main"

  depends_on "go" => :build

  def install
    # Build the main CLI tool
    system "go", "build", "-ldflags", "-s -w -X main.version=#{version}",
           "-o", bin/"mcp-validator", "./pkg/mcp/tools/mcp-validator"
    
    system "go", "build", "-ldflags", "-s -w -X main.version=#{version}",
           "-o", bin/"mcp-benchmark", "./pkg/mcp/tools/mcp-benchmark"
    
    # Build example servers
    system "go", "build", "-o", bin/"mcp-echo-server", "./pkg/mcp/examples/echo-server"
    system "go", "build", "-o", bin/"mcp-calculator", "./pkg/mcp/examples/calculator"
    system "go", "build", "-o", bin/"mcp-file-manager", "./pkg/mcp/examples/file-manager"
    system "go", "build", "-o", bin/"mcp-weather-service", "./pkg/mcp/examples/weather-service"
    
    # Install shell completions
    generate_completions_from_executable(bin/"mcp-validator", "completion")
    
    # Install documentation
    doc.install "README.md"
    doc.install "pkg/mcp/README.md" => "MCP_LIBRARY.md"
    doc.install Dir["docs/**/*.md"]
    
    # Install examples
    pkgshare.install "pkg/mcp/examples"
  end

  service do
    run [opt_bin/"mcp-echo-server"]
    keep_alive true
    log_path var/"log/mcp-go/echo-server.log"
    error_log_path var/"log/mcp-go/echo-server-error.log"
  end

  test do
    # Test validator
    assert_match "MCP Validator", shell_output("#{bin}/mcp-validator --version")
    
    # Test benchmark tool
    assert_match "MCP Benchmark", shell_output("#{bin}/mcp-benchmark --version")
    
    # Test echo server (start and stop)
    pid = fork do
      exec bin/"mcp-echo-server"
    end
    sleep 2
    
    begin
      # Test that server is running
      require "net/http"
      response = Net::HTTP.get_response(URI("http://localhost:8080/health"))
      assert_equal "200", response.code
    ensure
      Process.kill("TERM", pid)
      Process.wait(pid)
    end
  end
end