import { useEffect, useState, useCallback } from 'react'

interface Message {
  type: string
  data?: any
  msg?: string
}

export function useWebSocket(url: string) {
  const [data, setData] = useState<any>(null)
  const [error, setError] = useState<string | null>(null)
  const [ws, setWs] = useState<WebSocket | null>(null)

  useEffect(() => {
    const websocket = new WebSocket(url)

    websocket.onopen = () => {
      console.log('WebSocket connected')
      setError(null)
    }

    websocket.onmessage = (event) => {
      const message: Message = JSON.parse(event.data)
      setData(message)
    }

    websocket.onerror = (event) => {
      setError('WebSocket error')
    }

    setWs(websocket)

    return () => {
      websocket.close()
    }
  }, [url])

  const send = useCallback((message: any) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(message))
    }
  }, [ws])

  return { data, error, send }
}
