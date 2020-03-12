var path = window.location.pathname;
var branches = document.getElementById("branchSelect");
var pathStr = []

for (i = 0; i < branches.length; i++) {
    var branchValue = branches[i].value;
    pathSplit = path.split(branchValue);

    if (pathSplit.length == 2) {
        branches.value = branchValue
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

