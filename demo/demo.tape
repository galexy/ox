# SageOx Demo - Session Capture + Team Context Discovery
#
# Shows: ox login → ox init → ox status → Claude Code with /ox-session-start
#        → Second developer (Ajit) discovers team context via ox
#
# Uses mock commands in demo/mock/ for reproducible recording.
#
# Record:
#   vhs demo/demo.tape
#
# Output: demo/demo.gif

Output demo/demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Catppuccin Mocha"
Set Padding 20
Set Framerate 24
Set TypingSpeed 40ms

Hide

# Setup: mock commands on PATH, clean prompt
Type "export PS1='$ '"
Enter
Type@0 "export PATH=$(pwd)/demo/mock:$PATH"
Enter
Type "clear"
Enter

Show
Sleep 500ms

# Step 1: Login to SageOx
Type "ox login"
Enter
Sleep 6s

# Step 2: Initialize SageOx in the repo
Type "ox init"
Enter
Sleep 6s

# Step 3: Check project status
Type "ox status"
Enter
Sleep 5s

# Step 4: Launch Claude Code (as Ryan)
Type "claude"
Enter
Sleep 3s

# Step 5: Start a session
Type "/ox-session-start"
Enter
Sleep 4s

# Hold on session start confirmation, then exit
Sleep 2s
Type "/exit"
Enter
Sleep 1s

# Step 6: Switch to Ajit's perspective
Type "# Now Ajit (a different developer) opens Claude Code..."
Enter
Sleep 2s

Type "claude"
Enter
Sleep 3s

# Step 7: Ajit asks about team discussions
Type "Use ox status to get SageOx team context location that has discussions for this repo, understand the details of the conversation between me and ryan, and go into planning mode for that"
Enter
Sleep 6s

# Hold on the response
Sleep 3s
