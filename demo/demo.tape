# The obon demo, shot with foley: the real TUI on a real pty, no window,
# no screen capture. Two throwaway servers are lit off camera (one through
# a fake `claude` wrapper so ORIGIN shows an agent), then the reel moves:
# the board, the filter, the parent chain, the send off.

Output assets/demo.gif

Require obon
Set Shell bash
Set Width 940
Set Height 560
Set FontSize 15
Set Framerate 30
Set TypingSpeed 25ms
Set WaitTimeout 30s

Env PS1 "obon ❯ "

# setup (off camera): fake agent wrapper + two lingering spirits
Hide
Type@0ms `mkdir -p /tmp/obon-demo && printf '#!/bin/bash\n"$@" &\nwait\n' > /tmp/obon-demo/claude && chmod +x /tmp/obon-demo/claude && cd /tmp/obon-demo && echo "spirit one" > index.html && PATH=/tmp/obon-demo:$PATH claude python3 -m http.server 8946 --bind 127.0.0.1 >/dev/null 2>&1 & sleep 0.3; cd /tmp/obon-demo && python3 -m http.server 8947 --bind 127.0.0.1 >/dev/null 2>&1 & sleep 0.6; clear`
Enter
Sleep 1s
Show

Type "obon"
Enter
Sleep 4s

Type "j"
Sleep 0.4s
Type "j"
Sleep 1.5s

Type "/8946"
Enter
Sleep 1.5s
Enter
Sleep 4s
Escape
Sleep 0.8s

Type "x"
Sleep 2.5s
Type "y"
Sleep 3.5s

Escape
Sleep 0.8s
Escape
Sleep 1s

Hide
Type@0ms `pkill -f "http.server 8947" >/dev/null 2>&1; true`
Enter
Show
