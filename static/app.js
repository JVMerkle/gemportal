function onSubmit() {
    var url = document.getElementById("idUrl");

    if(url.value.startsWith("gemini://")) {
        url.value = url.value.slice(9)
    }

    var actionSrc = BASE_HREF + url.value;
    var insecure = document.getElementById("idInsecure");

    if(insecure.checked) {
        window.location.href = actionSrc + "?insecure=on";
    } else {
        window.location.href = actionSrc;
    }

    return false;
}