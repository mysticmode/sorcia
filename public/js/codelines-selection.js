document.addEventListener('DOMContentLoaded', (event) => {

    document.querySelectorAll('pre code').forEach((block) => {
        hljs.highlightBlock(block);
        hljs.initLineNumbersOnLoad();
    });
});

window.onload = function() {
    // Add ID attribute to hljs elements/codelines.
    function hljsOnLoadAddID() {
        var hljs = document.querySelectorAll('pre > code > table > tbody > tr');
        if (hljs.length) {
            for (i = 0; i < hljs.length; ++i) {
                var lineNumber = hljs[i].firstChild.getAttribute('data-line-number');
                hljs[i].firstChild.setAttribute("id", "L"+lineNumber);
            }
        } else {
            setTimeout(hljsOnLoadAddID, 15);
        }
    }

    hljsOnLoadAddID();

    function hljsOnLoadScrollIntoView() {
        var hljs = document.querySelectorAll('pre > code > table > tbody > tr');
        if (hljs.length) {
            var url = window.location.href;
            var previousID, currentID;

            if (url.split('#')[1]) {
                if (url.split('#')[1].split('-')[1]) {
                    previousID = url.split('#')[1].split('-')[0];
                    currentID = url.split('#')[1].split('-')[1];
                    applyMultipleCodeLinesBg(previousID, currentID);

                    var previousNumber = parseInt(previousID.split('L')[1]);
                    var currentNumber = parseInt(currentID.split('L')[1]);
                    if (previousNumber < currentNumber) {
                        document.getElementById(previousID).scrollIntoView();
                    } else {
                        document.getElementById(currentID).scrollIntoView();
                    }
                } else {
                    currentID = url.split('#')[1];
                    applyCodeLineBg(currentID);

                    document.getElementById(currentID).scrollIntoView();
                }
            }
        } else {
            setTimeout(hljsOnLoadScrollIntoView, 15);
        }
    }

    hljsOnLoadScrollIntoView();

    window.onclick = function(e) {
        if (e.target.className == 'hljs-ln-n') {
            if (e.shiftKey) {
                var hljsLine = document.getElementsByClassName('hljs-ln-line');
                for (i = 0; i < hljsLine.length; i++) {
                    if(hljsLine[i].parentElement.className == "hljs-selection") {
                        var previousID = hljsLine[i].getAttribute('id');
                        var currentID = e.target.parentElement.getAttribute('id');
                        if (previousID == currentID) {
                            break;
                        } else {
                            applyMultipleCodeLinesBg(previousID, currentID);
                            // Update URL parameter to the combined lines ID
                            var url = window.location.href;
                            var newurl = url.split('#')[0] + '#' + previousID + '-' + currentID;
                            window.location.href = newurl;
                        }

                        applyMultipleCodeLinesBg(previousID, currentID);

                        break;
                    }
                }
            } else {
                var currentID = e.target.parentElement.getAttribute('id');
                applyCodeLineBg(currentID);
            }
        }
    }

    function applyCodeLineBg(currentID) {
        removeCodeLineBg();

        // Apply line background to the current ID element
        document.getElementById(currentID).parentElement.className = 'hljs-selection';

        // Update URL parameter to the Line ID
        var url = window.location.href;
        var newurl = url.split('#')[0] + '#' + currentID;
        window.location.href = newurl;
    }

    function applyMultipleCodeLinesBg(previousID, currentID) {
        removeCodeLineBg();

        var previousNumber = parseInt(previousID.split('L')[1]);
        var currentNumber = parseInt(currentID.split('L')[1]);

        if (previousNumber < currentNumber) {
            for (j = previousNumber; j <= currentNumber; j++) {
                document.getElementById('L'+j).parentElement.className = 'hljs-selection';
            }
        } else {
            for (j = previousNumber; j >= currentNumber; j--) {
                document.getElementById('L'+j).parentElement.className = 'hljs-selection';
            }
        }

        // Update URL parameter to the combined lines ID
        var url = window.location.href;
        var newurl = url.split('#')[0] + '#' + previousID + '-' + currentID;
        window.location.href = newurl;
    }

    function removeCodeLineBg() {
        // Remove previous line backgrounds
        var hljsLine = document.getElementsByClassName('hljs-ln-line');
        for (i = 0; i < hljsLine.length; ++i) {
            hljsLine[i].parentElement.classList.remove('hljs-selection');
        }
    }
};