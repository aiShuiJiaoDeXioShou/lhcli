#!/bin/sh
# lhcli 安装脚本。
#
# 用法：
#   curl -fsSL https://raw.githubusercontent.com/aiShuiJiaoDeXioShou/lhcli/main/install.sh | sh
#
# 环境变量：
#   VERSION       要安装的发布标签（默认：latest）
#   INSTALL_DIR   安装目录（默认：/usr/local/bin）

set -eu

REPO="aiShuiJiaoDeXioShou/lhcli"
BINARY="lhcli"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-latest}"

# err 打印错误信息并退出。
err() {
	echo "错误：$*" >&2
	exit 1
}

# need_cmd 检查命令是否存在。
need_cmd() {
	command -v "$1" >/dev/null 2>&1 || err "缺少必需的命令：$1"
}

need_cmd uname
need_cmd curl
need_cmd tar
need_cmd mktemp
need_cmd awk

if command -v sha256sum >/dev/null 2>&1; then
	checksum_cmd="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
	checksum_cmd="shasum"
else
	err "缺少 SHA256 校验工具，请安装 sha256sum 或 shasum"
fi

os="$(uname -s)"
case "$os" in
	Linux) os="linux" ;;
	Darwin) os="darwin" ;;
	*) err "不支持的操作系统：$os（请从 GitHub Releases 手动安装）" ;;
esac

arch="$(uname -m)"
case "$arch" in
	x86_64 | amd64) arch="amd64" ;;
	arm64 | aarch64) arch="arm64" ;;
	*) err "不支持的架构：$arch" ;;
esac

if [ "$VERSION" = "latest" ]; then
	VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
		| grep '"tag_name"' \
		| head -n1 \
		| sed -E 's/.*"([^"]+)".*/\1/')"
	[ -n "$VERSION" ] || err "无法解析最新的发布版本"
fi

case "$VERSION" in
	"" | *[!A-Za-z0-9._-]*) err "版本标签包含非法字符：$VERSION" ;;
esac

ver="${VERSION#v}"
archive="${BINARY}_${ver}_${os}_${arch}.tar.gz"
url="https://github.com/${REPO}/releases/download/${VERSION}/${archive}"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "正在下载 ${url}"
curl -fsSL "$url" -o "${tmp}/${archive}"
curl -fsSL "https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt" -o "${tmp}/checksums.txt"

expected="$(awk -v name="$archive" '$2 == name || $2 == "*" name {print tolower($1)}' "${tmp}/checksums.txt")"
[ "${#expected}" -eq 64 ] || err "校验和文件中缺少唯一且有效的 ${archive} 条目"
case "$expected" in
	*[!0-9a-fA-F]*) err "SHA256 校验和格式无效" ;;
esac
if [ "$checksum_cmd" = "sha256sum" ]; then
	actual="$(sha256sum "${tmp}/${archive}" | awk '{print $1}')"
else
	actual="$(shasum -a 256 "${tmp}/${archive}" | awk '{print $1}')"
fi
[ "$actual" = "$expected" ] || err "SHA256 校验失败，已停止安装，现有程序未修改"
echo "SHA256 校验通过"

tar -xzf "${tmp}/${archive}" -C "${tmp}" "$BINARY"
[ -f "${tmp}/${BINARY}" ] || err "压缩包中不包含 ${BINARY} 二进制文件"
[ ! -L "${tmp}/${BINARY}" ] || err "压缩包中的二进制文件不能是符号链接"
chmod +x "${tmp}/${BINARY}"

if [ ! -d "$INSTALL_DIR" ]; then
	mkdir -p "$INSTALL_DIR" 2>/dev/null || {
		need_cmd sudo
		sudo mkdir -p "$INSTALL_DIR"
	}
fi

if [ -w "$INSTALL_DIR" ]; then
	mv "${tmp}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
	need_cmd sudo
	echo "正在安装到 ${INSTALL_DIR}（可能需要输入密码）"
	sudo mv "${tmp}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

echo "已安装 ${BINARY} ${VERSION} 到 ${INSTALL_DIR}/${BINARY}"
"${INSTALL_DIR}/${BINARY}" version || true
