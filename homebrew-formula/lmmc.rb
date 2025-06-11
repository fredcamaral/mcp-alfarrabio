class Lmmc < Formula
  desc "Lerian MCP Memory CLI - Intelligent multi-repository task management with AI-powered insights"
  homepage "https://github.com/lerianstudio/lerian-mcp-memory"
  url "https://github.com/lerianstudio/lerian-mcp-memory/archive/v0.1.0.tar.gz"
  sha256 "f5d7d8070e3ee56db0013cb0637e4bd2dcf03fabe7eca17fd445fd734581c1fb"
  license "MIT"
  head "https://github.com/lerianstudio/lerian-mcp-memory.git", branch: "main"

  depends_on "go" => :build

  def install
    cd "cli" do
      system "go", "build", *std_go_args(ldflags: "-s -w"), "-o", bin/"lmmc", "./cmd/lmmc"
    end

    # Install man page
    man1.install "docs/lmmc.1" if File.exist?("docs/lmmc.1")
    
    # Install shell completions
    generate_completions_from_executable(bin/"lmmc", "completion")
  end

  service do
    run [opt_bin/"lmmc", "tui", "--mode", "dashboard"]
    keep_alive false
    log_path var/"log/lmmc.log"
    error_log_path var/"log/lmmc.log"
  end

  test do
    system "#{bin}/lmmc", "version"
    system "#{bin}/lmmc", "help"
    
    # Test basic functionality
    system "#{bin}/lmmc", "add", "test task"
    output = shell_output("#{bin}/lmmc", "list")
    assert_match "test task", output
  end
end