# Example Files

The `examples` directory contains sample assets that help you run the project in different environments.

- `config.yaml` – minimal configuration file for running the service.
- `systemd/project-template.service` – sample unit file to manage the service under systemd.
- `systemd/project-template.socket` – socket unit for systemd activation.
- `../schema/item.sql` – SQL script to create the `items` table with example data.

Copy these files to a suitable location such as `/etc/project-template/` and adjust any paths as needed. Enable the socket unit to use systemd socket activation and set the `SYSTEMD_SOCKET_ACTIVATION` environment variable to `true`.

The systemd units are referenced from the main [Systemd Support](../README.md#systemd-support) section of the root README.
