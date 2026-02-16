# SageOx + AI Coding Agent Demo
#
# This demonstrates ox review generating infrastructure findings
# and an AI coding agent filing beads tasks automatically.
#
# Prerequisites:
#   1. Run ./demo/setup.sh first
#   2. Claude Code installed (claude command) - or any AI coding agent
#   3. beads installed (bd command)
#
# Record with: vhs demo/claude-demo.tape

Output demo/claude-demo.gif

Set Shell "bash"
Set FontSize 13
Set Width 1400
Set Height 900
Set Theme "Catppuccin Mocha"
Set Padding 20
Set Framerate 24

# Set minimal prompt to hide environment details
Hide
Type "export PS1='% '"
Enter
Type "rm -rf /tmp/sagebox/inbox-zero"
Enter
Type "mkdir -p /tmp/sagebox/inbox-zero"
Enter
Type "cd /tmp/sagebox/inbox-zero"
Enter
Type "clear"
Enter
Show
Sleep 500ms

# Title
Type "# SageOx + AI Coding Agent: Multiplayer Infrastructure Review"
Enter
Sleep 1.5s

Type "# Maya (Auth Team) starts her morning with an infra review..."
Enter
Sleep 1s

# Start Claude Code with the review prompt
Type "claude"
Enter
Sleep 3s

# Send the ox review command
Type "First create an epic called 'Infrastructure Modernization Q1' with bd create --type epic. Then run ox review and create 5 beads tasks for the most important infrastructure improvements. Link each task to the epic using bd dep."
Enter
Sleep 20s

# Let Claude work...
Sleep 15s

# Exit Claude
Type "/exit"
Enter
Sleep 1s

# Show the created epic and tasks
Type "# Epic and tasks created by Maya's AI agent:"
Enter
Sleep 500ms

Type "bd show inbox-zero-1"
Enter
Sleep 2s

Type "bd list"
Enter
Sleep 2s

# Show ready tasks
Type "# Jordan (Payments Team) checks what's ready to work on..."
Enter
Sleep 1s

Type "bd ready"
Enter
Sleep 2s

# Show stats
Type "# Team progress visible to all:"
Enter
Sleep 500ms

Type "bd stats"
Enter
Sleep 1.5s

# Final message
Type "# Three engineers, three AI agents, one shared backlog"
Enter
Sleep 1.5s

Type "# Riley's Windsurf agent can pick up any of these tasks next"
Enter
Sleep 2s

Type "# Learn more: https://sageox.ai"
Enter
Sleep 2s
