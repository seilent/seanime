import { logger } from "@/lib/helpers/debug"

const audioBoostLog = logger("SEA MEDIA PLAYER AUDIO BOOST")

type BoostGraph = {
    ctx: AudioContext
    source: MediaElementAudioSourceNode
    gain: GainNode
    limiter: DynamicsCompressorNode
}

const graphs = new WeakMap<HTMLMediaElement, BoostGraph>()

function createGraph(element: HTMLMediaElement): BoostGraph | null {
    const w = window as unknown as { AudioContext?: typeof AudioContext; webkitAudioContext?: typeof AudioContext }
    const Ctor = w.AudioContext ?? w.webkitAudioContext
    if (!Ctor) return null

    try {
        const ctx = new Ctor()
        const source = ctx.createMediaElementSource(element)

        const gain = ctx.createGain()
        gain.gain.value = 1

        const limiter = ctx.createDynamicsCompressor()
        limiter.threshold.value = -1
        limiter.knee.value = 0
        limiter.ratio.value = 20
        limiter.attack.value = 0.001
        limiter.release.value = 0.01

        const graph: BoostGraph = { ctx, source, gain, limiter }
        graphs.set(element, graph)
        return graph
    }
    catch (e) {
        audioBoostLog.error("Failed to create audio boost graph", e)
        return null
    }
}

export function applyAudioBoost(element: HTMLMediaElement | null, boost: number) {
    if (!element) return

    let graph = graphs.get(element)
    if (!graph) {
        if (boost <= 1) return
        graph = createGraph(element)
        if (!graph) return
    }

    if (graph.ctx.state === "suspended") {
        void graph.ctx.resume().catch(() => {})
    }

    graph.source.disconnect()
    graph.gain.disconnect()
    graph.limiter.disconnect()

    if (boost > 1) {
        graph.gain.gain.value = boost
        graph.source.connect(graph.gain)
        graph.gain.connect(graph.limiter)
        graph.limiter.connect(graph.ctx.destination)
    } else {
        graph.source.connect(graph.ctx.destination)
    }
}

export function resumeAudioBoost(element: HTMLMediaElement | null) {
    if (!element) return
    const graph = graphs.get(element)
    if (graph && graph.ctx.state === "suspended") {
        void graph.ctx.resume().catch(() => {})
    }
}
