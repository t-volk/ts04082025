# Test Server
A test web server that aims to download files from links, archive them, and provide the user to archives with links.
## Implemented
- [x] user interface is accessible through a browser
- [x] adding tasks (no more than 3)
- [x] adding up to 3 files per task
- [x] downloading files by the server (jpeg and pdf types)
## In future
- [ ] generation of zip archives and making available for download
- [ ] fine-tuning limits and network port
## Running server
It has to be done in order:
1) Starting the server from terminal
```
go run .
```
2) Launching tasks in the browser
```
http://127.0.0.1:8080
```
