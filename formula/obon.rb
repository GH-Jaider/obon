# Homebrew formula stub, goreleaser overwrites this on release.
# Tap: GH-Jaider/homebrew-obon
class Obon < Formula
  desc "Every dev server your agents summoned, still lingering on your ports"
  homepage "https://github.com/GH-Jaider/obon"
  version "0.1.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/GH-Jaider/obon/releases/download/v#{version}/obon_#{version}_darwin_arm64.tar.gz"
    else
      url "https://github.com/GH-Jaider/obon/releases/download/v#{version}/obon_#{version}_darwin_amd64.tar.gz"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/GH-Jaider/obon/releases/download/v#{version}/obon_#{version}_linux_arm64.tar.gz"
    else
      url "https://github.com/GH-Jaider/obon/releases/download/v#{version}/obon_#{version}_linux_amd64.tar.gz"
    end
  end

  def install
    bin.install "obon"
  end

  def caveats
    <<~EOS
      Run `obon` to open the board. `obon clean --older-than 2h` decants
      listeners younger than two hours (tōrō nagashi).
    EOS
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/obon version")
  end
end
