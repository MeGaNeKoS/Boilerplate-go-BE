# Systemd Units

This folder contains service and socket unit files that show how to run the project under systemd. The `project-template.service` unit runs the main binary, while the `project-template-supervisor.service` coordinates graceful restarts when enabled.

To use the supervisor approach, enable `project-template.socket` and uncomment the environment variables in `project-template.service` as noted in the file.
