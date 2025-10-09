'use client'

import React, { useState, useEffect, useRef } from 'react'
import { Button } from '@/components/ui/button'
import { X } from 'lucide-react'

interface LogEntry {
  time: string
  level: 'INFO' | 'WARNING' | 'ERROR' | 'SUCCESS'
  message: string
  component?: string
}

const mockLogs: LogEntry[] = [
  { time: '11:00:01', level: 'INFO', message: 'Starting backup job for', component: 'Backup Engine' },
  { time: '11:00:02', level: 'INFO', message: 'Connecting to vCenter serv', component: 'VMware API' },
  { time: '11:00:03', level: 'INFO', message: 'Snapshot created successfu', component: 'VMware API' },
  { time: '11:00:04', level: 'INFO', message: 'Transferring data: 25% c', component: 'NBD Transfer' },
  { time: '11:00:05', level: 'WARNING', message: 'Network latency detec ted, adjusting buffer size', component: 'NBD Transfer' },
  { time: '11:00:06', level: 'INFO', message: 'Transferring data: 50% c', component: 'NBD Transfer' },
  { time: '11:00:07', level: 'INFO', message: 'Transferring data: 75% c', component: 'NBD Transfer' },
  { time: '11:00:08', level: 'INFO', message: 'Transferring data: 100%', component: 'NBD Transfer' },
  { time: '11:00:09', level: 'INFO', message: 'Verifying backup in', component: 'Validation Engine' },
  { time: '11:00:10', level: 'INFO', message: 'Backup completed succes sfully', component: 'Backup Engine' },
]

const logLevelColors = {
  INFO: 'text-blue-400',
  WARNING: 'text-yellow-400',
  ERROR: 'text-red-400',
  SUCCESS: 'text-green-400',
}

interface JobLogsDrawerProps {
  isOpen: boolean
  onClose: () => void
}

export function JobLogsDrawer({ isOpen, onClose }: JobLogsDrawerProps) {
  const [logs, setLogs] = useState<LogEntry[]>(mockLogs)
  const [autoScroll, setAutoScroll] = useState(true)
  const [filter, setFilter] = useState<'All' | 'INFO' | 'WARNING' | 'ERROR'>('All')
  const [width, setWidth] = useState(400)
  const [isResizing, setIsResizing] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)

  // Load state from localStorage on mount
  useEffect(() => {
    const savedWidth = localStorage.getItem('jobLogsWidth')
    if (savedWidth) {
      const parsedWidth = parseInt(savedWidth)
      if (parsedWidth >= 300 && parsedWidth <= 600) {
        setWidth(parsedWidth)
      }
    }
  }, [])

  // Save width to localStorage when it changes
  useEffect(() => {
    localStorage.setItem('jobLogsWidth', width.toString())
  }, [width])

  // Auto-scroll to bottom when new logs arrive
  useEffect(() => {
    if (autoScroll && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [logs, autoScroll])

  const filteredLogs = filter === 'All'
    ? logs
    : logs.filter(log => log.level === filter)

  const handleClear = () => {
    setLogs([])
  }

  const handleMouseDown = (e: React.MouseEvent) => {
    setIsResizing(true)
    const startX = e.clientX
    const startWidth = width

    const handleMouseMove = (moveEvent: MouseEvent) => {
      const deltaX = startX - moveEvent.clientX
      const newWidth = Math.max(300, Math.min(600, startWidth + deltaX))
      setWidth(newWidth)
    }

    const handleMouseUp = () => {
      setIsResizing(false)
      document.removeEventListener('mousemove', handleMouseMove)
      document.removeEventListener('mouseup', handleMouseUp)
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
    }

    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)
    document.body.style.cursor = 'ew-resize'
    document.body.style.userSelect = 'none'
  }

  return (
    <>
      {/* Drawer */}
      <div
        className="fixed top-0 bottom-0 right-0 bg-gray-900 border-l border-gray-700 flex flex-col z-50 shadow-2xl"
        style={{
          width: isOpen ? `${width}px` : '0px',
          transition: isResizing ? 'none' : 'width 300ms ease-in-out'
        }}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-gray-700 bg-gray-800/50 shrink-0">
          <div className="flex items-center gap-2">
            <h3 className="text-sm font-semibold text-white">Job Logs</h3>
            <span className="px-2 py-0.5 text-xs bg-green-500/20 text-green-400 rounded flex items-center gap-1">
              <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
              Live
            </span>
          </div>

          <div className="flex items-center gap-2">
            <select
              value={filter}
              onChange={(e) => setFilter(e.target.value as any)}
              className="text-xs bg-gray-700 border-gray-600 text-white rounded px-2 py-1"
            >
              <option value="All">All</option>
              <option value="INFO">Info</option>
              <option value="WARNING">Warning</option>
              <option value="ERROR">Error</option>
            </select>

            <button
              onClick={() => setAutoScroll(!autoScroll)}
              className={`text-xs px-2 py-1 rounded ${
                autoScroll
                  ? 'bg-blue-500/20 text-blue-400'
                  : 'bg-gray-700 text-gray-400'
              }`}
            >
              Auto-scroll
            </button>

            <button
              onClick={handleClear}
              className="text-xs px-2 py-1 rounded bg-gray-700 text-gray-400 hover:bg-gray-600"
            >
              <X className="h-3 w-3" />
            </button>

            <button
              onClick={onClose}
              className="text-xs px-2 py-1 rounded bg-gray-700 text-gray-400 hover:bg-gray-600"
            >
              ✕
            </button>
          </div>
        </div>

        {/* Logs Display */}
        <div
          ref={scrollRef}
          className="flex-1 overflow-auto p-2 space-y-0.5"
        >
          {filteredLogs.length === 0 ? (
            <div className="flex items-center justify-center h-full text-gray-500 text-sm">
              No logs to display
            </div>
          ) : (
            filteredLogs.map((log, index) => (
              <div
                key={index}
                className="px-2 py-1 font-mono text-xs hover:bg-gray-800/30 cursor-pointer rounded"
              >
                <span className="text-gray-500">{log.time}</span>
                {' '}
                <span className={`font-semibold ${logLevelColors[log.level]}`}>
                  [{log.level}]
                </span>
                {' '}
                <span className="text-gray-300">{log.message}</span>
                {log.component && (
                  <>
                    {' '}
                    <span className="text-blue-300">{log.component}</span>
                  </>
                )}
              </div>
            ))
          )}
        </div>
      </div>

      {/* Resize Handle */}
      {isOpen && (
        <div
          onMouseDown={handleMouseDown}
          className={`fixed top-0 bottom-0 w-1 bg-gray-700 hover:bg-blue-500 cursor-ew-resize z-40 transition-colors ${
            isResizing ? 'bg-blue-500' : ''
          }`}
          style={{ right: width }}
        />
      )}
    </>
  )
}
