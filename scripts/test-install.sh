#!/bin/sh
# 使用本地下载替身验证安装校验，不访问网络或系统安装目录。
set -eu

project_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
fixture_dir="$(mktemp -d)"
trap 'rm -rf "$fixture_dir"' EXIT
export fixture_dir
mkdir -p "$fixture_dir/bin" "$fixture_dir/assets" "$fixture_dir/payload"

cat > "$fixture_dir/payload/lhcli" <<'EOF'
#!/bin/sh
echo "测试二进制"
EOF
archive="lhcli_0.0.0_linux_amd64.tar.gz"
tar -czf "$fixture_dir/assets/$archive" -C "$fixture_dir/payload" lhcli
if command -v sha256sum >/dev/null 2>&1; then
	checksum="$(sha256sum "$fixture_dir/assets/$archive" | awk '{print $1}')"
else
	checksum="$(shasum -a 256 "$fixture_dir/assets/$archive" | awk '{print $1}')"
fi
printf '%s  %s\n' "$checksum" "$archive" > "$fixture_dir/assets/checksums.txt"

cat > "$fixture_dir/bin/uname" <<'EOF'
#!/bin/sh
case "$1" in
	-s) echo Linux ;;
	-m) echo x86_64 ;;
esac
EOF
cat > "$fixture_dir/bin/curl" <<'EOF'
#!/bin/sh
cp "$fixture_dir/assets/${2##*/}" "$4"
EOF
chmod +x "$fixture_dir/bin/uname" "$fixture_dir/bin/curl"

PATH="$fixture_dir/bin:$PATH" VERSION=v0.0.0 INSTALL_DIR="$fixture_dir/install" sh "$project_dir/install.sh"
[ -x "$fixture_dir/install/lhcli" ] || { echo "安装产物不可执行" >&2; exit 1; }
printf '%s\n' "保留原程序" > "$fixture_dir/install/lhcli"
printf '%064d  %s\n' 0 "$archive" > "$fixture_dir/assets/checksums.txt"
if PATH="$fixture_dir/bin:$PATH" VERSION=v0.0.0 INSTALL_DIR="$fixture_dir/install" sh "$project_dir/install.sh"; then
	echo "错误：校验失败时仍然完成安装" >&2
	exit 1
fi
[ "$(cat "$fixture_dir/install/lhcli")" = "保留原程序" ] || { echo "错误：校验失败覆盖了原程序" >&2; exit 1; }
echo "安装脚本自检通过"
