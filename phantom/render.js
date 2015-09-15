var webPage = require('webpage');
var system = require('system');
var fs = require("fs");

var args = system.args;
var page = webPage.create();


if (args.length !== 4) {
	console.log("Usage: phantomjs render.js TIMEOUT URL FILEPATH");
	phantom.exit(1);
}

var timeout = parseInt(args[1])
var url = args[2];
var filePath = args[3];

if (timeout < 1) {
	console.log("Invalid timeout format, integer expected");
	phantom.exit(1);
}

console.log("Opening page " + url);
page.open(url, function(status) {
	// setTimeout is a workaround for "unsafe javascript attempt errors"
	// See https://github.com/ariya/phantomjs/issues/12697.
	setTimeout(function() {
		if (status == "fail") {
			console.log("Request failed");
			phantom.exit(1);
		}

		var start = new Date();

		setInterval(function() {
			var data = page.evaluate(function() {
				if ('serverRenderer' in window) {
					return window.serverRenderer;
				} else {
					return "";
				}
			});

			if (data == "") {
				var secondsTaken = (new Date().getTime() - start.getTime()) / 1000; 
				if (secondsTaken >= timeout) {
					console.log("Page did not report success within timeout");
					phantom.exit(1);
				} 
			} else {
				console.log("Saving content to " + filePath);
				var content = "<!-- http_status_code=" + data.status + " -->\n\n" + page.content;
				fs.write(filePath, content, 'w')
			  phantom.exit();
			}
		}, 100);
	}, 5);
});
