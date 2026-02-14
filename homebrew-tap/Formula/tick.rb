class Tick < Formula
  desc "CLI tool for tick"
  homepage "https://github.com/leeovery/tick"
  version "VERSION"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/leeovery/tick/releases/download/v#{version}/tick_#{version}_darwin_arm64.tar.gz"
      sha256 "SHA256_DARWIN_ARM64"
    elsif Hardware::CPU.intel?
      url "https://github.com/leeovery/tick/releases/download/v#{version}/tick_#{version}_darwin_amd64.tar.gz"
      sha256 "SHA256_DARWIN_AMD64"
    end
  end

  def install
    bin.install "tick"
  end

  test do
    assert_match "tick", shell_output("#{bin}/tick")
  end
end
