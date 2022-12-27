# file-share-cli

A quick way to share public files with friends.

I built this to get around Discord's file size limit.

Visiting the home path `/` returns a copy-able shell function with a dynamic URL like this:

```bash
# if you are me, copy this to your ~/.bashrc
# and use it like this: share somefile.txt
# and a download link will be echoed
# (don't forget to replace user/pass)
function share () {
	curl -u user:pass -F "file=@$1" https://my-url-at.railway.app/upload
}
```

You can then upload files from your terminal and get a download link.

Note: dragging files into the terminal on macOS pastes the file's path!

## Dev

Start the server.

```bash
AUTH_USERNAME=a AUTH_PASSWORD=b go run .
```

Visit `localhost:4000` and copy the bash snippet.

## Deploy

Deployable via [Railway](https://railway.app).

Set environmental variables `AUTH_USERNAME` and `AUTH_PASSWORD`.
