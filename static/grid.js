const h = 24
const w = 9
const textMax = 115
const textW = 120
let gridH = h
let gridW = w
let gridTextMax = textMax
let gridTextW = textW
let scale = 1

function fix_dpi(id) {
    let canvas = document.getElementById(id),
        ctx = canvas.getContext('2d'),
        dpi = window.devicePixelRatio;
    gridH = h * dpi.valueOf()
    gridW = w * dpi.valueOf()
    gridTextMax = textMax * dpi.valueOf()
    gridTextW = textW * dpi.valueOf()
    //console.log("scaling is "+dpi.valueOf())
    let style = {
        height() {
            return +getComputedStyle(canvas).getPropertyValue('height').slice(0,-2);
        },
        width() {
            return +getComputedStyle(canvas).getPropertyValue('width').slice(0,-2);
        }
    }
    canvas.setAttribute('width', style.width() * dpi);
    canvas.setAttribute('height', style.height() * dpi);
    scale = dpi.valueOf()
}

function legend() {
    const l = document.getElementById("legend")
    l.height = scale * h * 1.2
    //const scale = fix_dpi("legend")
    const ctx = l.getContext('2d')

    let offset = textW
    let grad = ctx.createLinearGradient(offset, 0, offset+gridW, gridH)
    grad.addColorStop(0, 'rgb(123,255,66)');
    grad.addColorStop(0.3, 'rgb(240,255,128)');
    grad.addColorStop(0.8, 'rgb(169,250,149)');
    ctx.fillStyle = grad
    ctx.fillRect(offset, 0, gridW, gridH)
    ctx.font = `${scale * 14}px sans-serif`
    ctx.fillStyle = 'grey'
    offset += gridW + gridW/2
    ctx.fillText("proposer",offset, gridH/1.2)

    offset += 65 * scale
    grad = ctx.createLinearGradient(offset, 0, offset+gridW, gridH)
    grad.addColorStop(0, 'rgba(0,0,0,0.2)');
    ctx.fillStyle = grad
    ctx.fillRect(offset, 0, gridW, gridH)
    ctx.fillStyle = 'grey'
    offset += gridW + gridW/2
    ctx.fillText("signed",offset, gridH/1.2)

    offset += 50 * scale
    grad = ctx.createLinearGradient(offset, 0, offset+gridW, gridH)
    grad.addColorStop(0, '#85c0f9');
    grad.addColorStop(0.7, '#85c0f9');
    grad.addColorStop(1, '#0b2641');
    ctx.fillStyle = grad
    ctx.fillRect(offset, 0, gridW, gridH)
    offset += gridW + gridW/2
    ctx.fillStyle = 'grey'
    ctx.fillText("miss/precommit",offset, gridH/1.2)

    offset += 110 * scale
    grad = ctx.createLinearGradient(offset, 0, offset+gridW, gridH)
    grad.addColorStop(0, '#381a34');
    grad.addColorStop(0.2, '#d06ec7');
    grad.addColorStop(1, '#d06ec7');
    ctx.fillStyle = grad
    ctx.fillRect(offset, 0, gridW, gridH)
    offset += gridW + gridW/2
    ctx.fillStyle = 'grey'
    ctx.fillText("miss/prevote", offset, gridH/1.2)

    offset += 90 * scale
    grad = ctx.createLinearGradient(offset, 0, offset+gridW, gridH)
    grad.addColorStop(0, '#8e4b26');
    grad.addColorStop(0.4, 'darkorange');
    ctx.fillStyle = grad
    ctx.fillRect(offset, 0, gridW, gridH)
    ctx.beginPath();
    ctx.moveTo(offset + 1, gridH-2-gridH/2);
    ctx.lineTo(offset + 4 + gridW / 4, gridH-1-gridH/2);
    ctx.closePath();
    ctx.strokeStyle = 'white'
    ctx.stroke();
    offset += gridW + gridW/2
    ctx.fillStyle = 'grey'
    ctx.fillText("missed", offset, gridH/1.2)

    offset += 59 * scale
    grad = ctx.createLinearGradient(offset, 0, offset+gridW, gridH)
    grad.addColorStop(0, 'rgba(127,127,127,0.3)');
    ctx.fillStyle = grad
    ctx.fillRect(offset, 0, gridW, gridH)
    offset += gridW + gridW/2
    ctx.fillStyle = 'grey'
    ctx.fillText("no data", offset, gridH/1.2)
}

function drawSeries(multiStates) {
    const canvas = document.getElementById("canvas")
    canvas.height = ((12*gridH*multiStates.Status.length)/10) + 30
    fix_dpi("canvas")
    if (canvas.getContext) {
        const ctx = canvas.getContext('2d')
        ctx.font = `${scale * 16}px sans-serif`

        let crossThrough = false
        for (let j = 0; j < multiStates.Status.length; j++) {

            ctx.fillStyle = 'white'
            ctx.fillText(multiStates.Status[j].name, 5, (j*gridH)+(gridH*2)-6, gridTextMax)

            for (let i = 0; i < multiStates.Status[j].blocks.length; i++) {
                crossThrough = false
                const grad = ctx.createLinearGradient((i*gridW)+gridTextW, (gridH*j), (i * gridW) + gridW +gridTextW, (gridH*j))
                switch (multiStates.Status[j].blocks[i]) {
                    case 4: // proposed
                        grad.addColorStop(0, 'rgb(123,255,66)');
                        grad.addColorStop(0.3, 'rgb(240,255,128)');
                        grad.addColorStop(0.8, 'rgb(169,250,149)');
                        break
                    case 3: // signed
                        if (j % 2 === 0) {
                            grad.addColorStop(0, 'rgba(0,0,0,0.4)');
                            grad.addColorStop(0.9, 'rgba(0,0,0,0.4)');
                        } else {
                            grad.addColorStop(0, 'rgba(0,0,0,0.1)');
                            grad.addColorStop(0.9, 'rgba(0,0,0,0.1)');
                        }
                        grad.addColorStop(1, 'rgb(186,186,186)');
                        break
                    case 2: // precommit not included
                        grad.addColorStop(0, '#85c0f9');
                        grad.addColorStop(0.8, '#85c0f9');
                        grad.addColorStop(1, '#0b2641');
                        break
                    case 1: // prevote not included
                        grad.addColorStop(0, '#381a34');
                        grad.addColorStop(0.2, '#d06ec7');
                        grad.addColorStop(1, '#d06ec7');
                        break
                    case 0: // missed
                        grad.addColorStop(0, '#c15600');
                        crossThrough = true
                        break
                    default:
                        grad.addColorStop(0, 'rgba(127,127,127,0.3)');
                }
                ctx.fillStyle = grad
                ctx.fillRect((i*gridW)+gridTextW, gridH+(gridH*j), gridW, gridH)
                ctx.beginPath();

                // line between rows
                ctx.moveTo((i*gridW)-gridW+gridTextW, 2*gridH+(gridH*j)-0.5)
                ctx.lineTo((i*gridW*2)+gridTextW, 2*gridH+(gridH*j)-0.5);
                ctx.closePath();
                ctx.strokeStyle = 'rgb(51,51,51)'
                ctx.strokeWidth = '5px;'
                ctx.stroke();

                // visual differentiation for missed blocks
                if (crossThrough) {
                    ctx.beginPath();
                    ctx.moveTo((i * gridW) + gridTextW + 1 + gridW / 4, (gridH*j) + (gridH * 2) - gridH / 2);
                    ctx.lineTo((i * gridW) + gridTextW + gridW - (gridW / 4) - 1, (gridH*j) + (gridH * 2) - gridH / 2);
                    ctx.closePath();
                    ctx.strokeStyle = 'white'
                    ctx.stroke();
                }
            }
        }
    }
}