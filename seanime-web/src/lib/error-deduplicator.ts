interface ErrorEntry {
	message: string
	timestamp: number
	count: number
}

export class ErrorDeduplicator {
	private errors: Map<string, ErrorEntry>
	private dedupeWindowMs: number
	private maxCacheSize: number

	constructor(dedupeWindowMs: number = 5000, maxCacheSize: number = 100) {
		this.errors = new Map()
		this.dedupeWindowMs = dedupeWindowMs
		this.maxCacheSize = maxCacheSize
	}

	shouldShowError(message: string): boolean {
		const now = Date.now()
		const key = this.normalizeError(message)

		// Clean old errors
		this.cleanup(now)

		const existing = this.errors.get(key)

		if (existing && now - existing.timestamp < this.dedupeWindowMs) {
			// Same error within window - increment count but don't show
			existing.count++
			return false
		}

		// New error or expired - show it
		this.errors.set(key, {
			message,
			timestamp: now,
			count: 1
		})

		// Prevent unbounded growth
		if (this.errors.size > this.maxCacheSize) {
			this.removeOldest()
		}

		return true
	}

	getErrorCount(message: string): number {
		const key = this.normalizeError(message)
		return this.errors.get(key)?.count || 0
	}

	private normalizeError(message: string): string {
		// Normalize: lowercase, trim, remove extra whitespace
		return message.toLowerCase().trim().replace(/\s+/g, ' ')
	}

	private cleanup(now: number): void {
		for (const [key, entry] of this.errors.entries()) {
			if (now - entry.timestamp >= this.dedupeWindowMs) {
				this.errors.delete(key)
			}
		}
	}

	private removeOldest(): void {
		let oldestKey: string | null = null
		let oldestTime = Infinity

		for (const [key, entry] of this.errors.entries()) {
			if (entry.timestamp < oldestTime) {
				oldestTime = entry.timestamp
				oldestKey = key
			}
		}

		if (oldestKey) {
			this.errors.delete(oldestKey)
		}
	}

	clear(): void {
		this.errors.clear()
	}
}
