
async function loadState() {
   const enableLogs = await fetch("logsenabled", {
   //const enableLogs = await fetch("http://127.0.0.1:8888/logsenabled", {
        method: 'GET',
        mode: 'cors',
        cache: 'no-cache',
        credentials: 'same-origin',
        redirect: 'error',
        referrerPolicy: 'no-referrer'
    });
    let showLog
    try {
        showLog = await enableLogs.json()
    } catch(e) {
        console.log(e)
    }
    if (showLog.enabled === false) {
        document.getElementById("logContainer").hidden = true
    }
    //const response = await fetch("http://127.0.0.1:8888/state", {
    const response = await fetch("state", {
        method: 'GET',
        mode: 'cors',
        cache: 'no-cache',
        credentials: 'same-origin',
        redirect: 'error',
        referrerPolicy: 'no-referrer'
    });
    let initialState
    try {
        initialState = await response.json()
    } catch(e) {
        console.log(e)
    }
    updateTable(initialState)
    drawSeries(initialState)
    const logResponse = await fetch("logs", {
    //const logResponse = await fetch("http://127.0.0.1:8888/logs", {
        method: 'GET',
        mode: 'cors',
        cache: 'no-cache',
        credentials: 'same-origin',
        redirect: 'error',
        referrerPolicy: 'no-referrer'
    });
    try {
        initialState = await logResponse.json()
    } catch(e) {
        console.log(e)
    }
    for (let i = initialState.length-1; i >= 0; i--) {
        if (initialState[i].ts === 0) {
            addLogMsg("")
            continue
        }
        addLogMsg(`${new Date(initialState[i].ts*1000).toLocaleTimeString()} - ${initialState[i].msg}`)
    }
}

const blocks = new Map();
function updateTable(status) {
    for (let i = document.getElementById("statusTable").rows.length; i > 0; i--) {
        document.getElementById("statusTable").deleteRow(i-1)
    }
    const fade = `uk-animation-scale-up`
    for (let i = 0; i < status.Status.length; i++) {

        let alerts = "&nbsp;"
        if (status.Status[i].active_alerts > 0 || status.Status[i].last_error !== "") {
            if (status.Status[i].last_error !== "") {
                alerts = `
            <a href="#modal-center-${status.Status[i].name}" uk-toggle><span uk-icon='warning' uk-tooltip="${_.escape(status.Status[i].active_alerts)} active issues" style='color: darkorange'></span></a>
            <div id="modal-center-${_.escape(status.Status[i].name)}" class="uk-flex-top" uk-modal>
                <div class="uk-modal-dialog uk-modal-body uk-margin-auto-vertical uk-background-secondary">
                    <button class="uk-modal-close-default" type="button" uk-close></button>
                    <pre class=" uk-background-secondary" style="color: white">${_.escape(status.Status[i].last_error)}</pre>
                </div>
            </div>
            `
            } else {
                alerts = `<span uk-icon='warning' uk-tooltip="${_.escape(status.Status[i].active_alerts)} active issues" style='color: darkorange'></span>`
            }
        }

        let bonded = ""
        switch (true) {
            case status.Status[i].tombstoned:
                bonded = "<div class='uk-text-warning'><span uk-icon='ban'></span> <strong>Tombstoned</strong></div>"
                break
            case status.Status[i].jailed:
                bonded = "<span uk-icon='warning'></span> <strong>Jailed</strong>"
                break
            case status.Status[i].bonded:
                bonded = "<span uk-icon='check'></span>"
                break
            default:
                bonded = "<span uk-icon='minus-circle'></span> Not active"
        }

        let window = `<div class="uk-width-1-2" style="text-align: end">`
        if (status.Status[i].missed === 0 && status.Status[i].window === 0) {
            window += "error</div>"
        } else if (status.Status[i].missed === 0) {
            window += `100%</div>`
        } else {
            window += `${(100 - (status.Status[i].missed / status.Status[i].window) * 100).toFixed(2)}%</div>`
        }
        window += `<div class="uk-width-1-2">${_.escape(status.Status[i].missed)} / ${_.escape(status.Status[i].window)}</div>`
        
        let threshold = ""
        threshold += `<span class="uk-width-1-2">${100 * status.Status[i].min_signed_per_window}%</span>`;

        let nodes = `${_.escape(status.Status[i].healthy_nodes)} / ${_.escape(status.Status[i].nodes)}`
        if (status.Status[i].healthy_nodes < status.Status[i].nodes) {
            nodes = "<strong><span uk-icon='arrow-down' style='color: darkorange'></span>" + nodes + "</strong>"
        }

        let heightClass = ""
        if (blocks.get(status.Status[i].chain_id) !== status.Status[i].height){
            heightClass = fade
        }
        blocks.set(status.Status[i].chain_id, status.Status[i].height)

        let r=document.getElementById('statusTable').insertRow(i)
        r.insertCell(0).innerHTML = `<div>${alerts}</div>`
        r.insertCell(1).innerHTML = `<div>${_.escape(status.Status[i].name)} (${_.escape(status.Status[i].chain_id)})</div>`
        r.insertCell(2).innerHTML = `<div class="${heightClass}" style="font-family: monospace; color: #6f6f6f; text-align: start">${_.escape(status.Status[i].height)}</div>`
        if (status.Status[i].moniker === "not connected") {
            r.insertCell(3).innerHTML = `<div class="uk-text-warning">${_.escape(status.Status[i].moniker)}</div>`
            bonded = "unknown"
        } else {
            r.insertCell(3).innerHTML = `<div class='uk-text-truncate'>${_.escape(status.Status[i].moniker.substring(0,24))}</div>`
        }
        r.insertCell(4).innerHTML = `<div style="text-align: center">${bonded}</div>`
        r.insertCell(5).innerHTML = `<div uk-grid>${window}</div>`
        r.insertCell(6).innerHTML = `<div class="uk-text-center">${threshold}</div>`
        r.insertCell(7).innerHTML = `<div class="uk-text-center">${nodes}</div>`
    }
}

let logs = new Array(1);
function addLogMsg(str) {
    if (logs.length >= 256) {
        logs.pop()
    }
    logs.unshift(str)
    if (document.visibilityState !== "hidden") {
        document.getElementById("logs").innerText = logs.join("\n")
    }
}

function connect() {
    let wsProto = "ws://"
    if (location.protocol === "https:") {
        wsProto = "wss://"
    }
    const parse = function (event) {
        const msg = JSON.parse(event.data);
        if (msg.msgType === "log"){
            addLogMsg(`${new Date(msg.ts*1000).toLocaleTimeString()} - ${msg.msg}`)
        } else if (msg.msgType === "update" && document.visibilityState !== "hidden"){
            updateTable(msg)
            drawSeries(msg)
        }
        event = null
    }
    const socket = new WebSocket(wsProto + location.host + '/ws');
    //const socket = new WebSocket('ws://127.0.0.1:8888/ws');
    socket.addEventListener('message', function (event) {parse(event)});
    socket.onclose = function(e) {
        console.log('Socket is closed, retrying /ws ...', e.reason);
        addLogMsg('Socket is closed, retrying /ws ...' + e.reason)
        setTimeout(function() {
            connect();
        }, 3000);
    };
}
connect()