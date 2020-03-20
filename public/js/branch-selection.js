var path = window.location.pathname;
var branches = document.getElementById("branchSelect");
var pathStr = []

for (i = 0; i < branches.length; i++) {
    var branchValue = branches[i].value;
    pathSplit = path.split(branchValue);

    if (pathSplit.length == 2) {
        branches.value = branchValue;

        // Update next button link href
        var href = document.getElementById("repoPagination").getAttribute("href");
        var hrefFirst = href.split("?")[0].split("/");
        var hrefSecond = href.split("?")[1];

        var newHref = "";
        for (j = 0; j < (hrefFirst.length-1); j++) {
            newHref = newHref + hrefFirst[j] + "/";
        }
        newHref = newHref + branchValue + "?" + hrefSecond;

        document.getElementById("repoPagination").href = newHref;

        break;
    }
}

function branchChange(branch) {
    for (i = 0; i < branches.length; i++) {
        var branchValue = branches[i].value;
        pathSplit = path.split(branchValue);

        if (pathSplit.length == 2) {
            break;
        }
    }
    var pathStr = pathSplit[0] + branch + pathSplit[1];
    window.location.pathname = pathStr;
}