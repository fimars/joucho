# joucho

[inua](https://github.com/lyricat/inua) 的登录界面，也是 inua 的样子：
一个写给 [greetd](https://github.com/kennylevinsen/greetd) 的 Wayland
greeter，用 Go 和 GTK4 写成，用的是 shell 自己的绑定
（[puregotk](https://codeberg.org/puregotk/puregotk)），穿的也是 shell
自己的配色主题（washi、yoru、aojashin、bunko、denko、rinko、isekai、
suimen）。英文说明见 [README.md](README.md)。

名字取自「情緒」（情绪、じょうちょ，Jōcho）：`isekai` 主题画的是
KAMITSUBAKI STUDIO 的虚拟歌手
[ヰ世界情緒](https://kamitsubaki.jp/)（异世界情绪），而登录界面正是
机器和人见面的地方。这是一件粉丝概念作品，与歌手本人及
KAMITSUBAKI STUDIO 没有关联。

显示管理器该做的事由 greetd 来做（它很小，没有自己的界面，各发行版
都打包）；这个程序是它的 greeter：显示卡片，通过 greetd 的 IPC 协议
向 PAM 要密码，然后启动所选的会话。一个 kiosk 式的 Wayland 合成器
（通常是 [cage](https://github.com/cage-kiosk/cage)）把它全屏显示在
VT 上。

```text
greetd  ──PAM──▶  password
   │  starts
   ▼
cage ──▶ joucho ──GREETD_SOCK──▶ greetd
                               │ authenticated
                               ▼
                        inua-session (Hyprland + inua-shell)
```

## 构建与预览

需要 Go 1.26；运行时需要 GTK4 库（不用 cgo，也不需要头文件：puregotk
在运行时打开 libgtk-4.so）。

```sh
go build -o joucho .
```

在正在运行的会话里预览 —— 不需要 greetd，也不做真正的认证：

```sh
./joucho --demo
./joucho --demo --theme yoru --text-scale 125
```

`--list` 打印找到的用户和会话，然后退出。

| 参数 | 说明 |
| --- | --- |
| `--config PATH` | 设置文件（默认 `/etc/joucho/config.toml`） |
| `--theme NAME` | 配色主题，覆盖配置里的 `theme` |
| `--text-scale N` | 80 到 150，覆盖 `text_scale` |
| `--demo` | 在自己的会话里运行，不经过 greetd；不做真正的认证 |
| `--sessions-dir DIRS` | 到哪里找会话，冒号分隔 |
| `--list` | 打印找到的用户和会话后退出 |
| `--render FILE` | 把 isekai 构图画成 PNG 后退出 |

## 安装

```sh
sudo scripts/install.sh
```

脚本会在需要时安装软件包（`greetd`、`cage`），把程序复制到
`/usr/local/bin`，写入 `/etc/joucho/config.toml`，替换
`/etc/greetd/config.toml`（旧文件保留为 `config.toml.~1~`），停用
SDDM（或当前启用的显示管理器）并启用 greetd。脚本不会启动或停止任何
服务：屏幕上的会话照常运行。`--no-dnf` 跳过安装软件包这一步，适合
已经装好 greetd 和 cage 的机器。

想看登录界面，重启即可；或者切到 TTY（Ctrl+Alt+F3）：

```sh
sudo systemctl stop sddm && sudo systemctl start greetd
```

回到 SDDM：`sudo scripts/revert.sh`。

## 设置

`/etc/joucho/config.toml`，改动之前全是默认值（greeter 以 greetd
用户运行，读不到你的家目录）：

| 设置 | 默认值 | 说明 |
| --- | --- | --- |
| `theme` | `"washi"` | 内置主题（`washi`、`yoru`、`isekai`……），或配置旁边的 `themes/NAME.css` |
| `text_scale` | `100` | 文字大小，80 到 150；含义与 inua 的 `style.text_scale` 相同，只缩放字号 |
| `language` | `"auto"` | `auto`（跟随 LANG）、`en`、`zh_CN` |
| `user` | `""` | 默认选中的用户；留空则选本次开机最后一个登录的用户 |
| `session` | `"inua.desktop"` | 默认选中的会话，来自 `/usr/share/wayland-sessions` |
| `wallpaper` | `""` | 卡片背后的图片；留空则用主题的底色纹理，和 shell 的壁纸一致 |
| `font` | `""` | 窗口字体；留空则用内置字体栈，也就是 shell 自己的（Zen Kaku Gothic New、Noto Sans CJK JP/SC） |
| `power` | `true` | 显示休眠、重启和关机按钮（它们询问 logind；没有权限时会提示） |
| `clock_seconds` | `false` | 时钟是否显示秒 |

自己的主题放在 `/etc/joucho/themes/NAME.css`：用与 inua 相同的
`@define-color` 令牌，只写要改的那些，其余沿用它所基于的主题。可以
从 shell 的 `~/.config/inua/themes/`（或本仓库的 `assets/themes/`）
复制一个开始。

## isekai 主题

`theme = "isekai"` 不是换一套颜色那么简单。它把 greeter 画成一张
关于 ヰ世界情緒 的角色研究：纸与颗粒，右侧的夜之面板与「ヰ」，藤蔓与
朱红的线，眼之花，色板。卡片落在纸上、在「ヰ」的左边。构图是 fimars
自己画的（`decor.go`），以概念的 1600 × 1000 为单位；
`--demo --theme isekai` 按这个尺寸显示，`--render FILE` 则完全不需要
显示器。

这是一件粉丝概念作品。屏幕上的署名标注了角色设计者，并写明
「not affiliated」；仓库里没有任何官方美术资源，本项目与歌手本人及
KAMITSUBAKI STUDIO 没有关联。

## 说明

- greeter 以 greetd 用户运行，看不到你家目录里的字体、主题或图片。
  内置字体栈来自 shell，除了系统的 Noto CJK 不需要额外安装。如果你把
  `font = "Family Name"` 指向家目录里的字体，请把它装到系统里
  （`/usr/local/share/fonts`，然后 `fc-cache -f`）；找不到该字体时，
  greeter 会在 greetd 日志里说明，并使用字体栈里的下一个。
- 电源按钮调用 `org.freedesktop.login1`。Fedora 的 polkit 策略通常允许
  活动会话直接操作；如果不允许，greeter 会提示「没有权限。」，不会有
  别的动作。
- greeter 打印的内容由 greetd 记入日志：`journalctl -u greetd`（有些
  系统是 `/var/log/greetd.log`）。
- greeter 的账户只能读 `/etc/passwd` 和 `/usr/share/wayland-sessions`，
  读不到你的东西；密码通过 socket 交给 greetd，由 PAM 校验，greeter
  不留存任何密码。
- `--demo` 里的 portal 警告（「Cannot get portal …」）无害：真正的
  greeter 会话没有 portal。
- 如果 SELinux 下 cage 拿不到 DRM 设备，用 `journalctl -u greetd` 查看；
  greetd 的 PAM 会话通常会提供 seat 权限。

## 开发

```sh
go test ./...
go vet ./...
```

`--render FILE.png` 不需要显示器就能画出 isekai 构图，一个概念单位对应
一个像素，并标出卡片的占位，这样不用和合成器较劲就能检查、微调构图。

如何提交改动见 [CONTRIBUTING.md](CONTRIBUTING.md)，改动记录见
[CHANGELOG.md](CHANGELOG.md)。

## 许可证

joucho 使用 Apache License 2.0，见 `LICENSE`。它复制了
[inua](https://github.com/lyricat/inua)（同样是 Apache-2.0）的配色主题、
`scaleText` 函数和底色纹理画法，出处记录在 `NOTICE` 中。
