import React, { useEffect, useRef } from 'react'
import * as d3 from 'd3'

interface GraphNode {
  id: string
  name: string
  type: string
}

interface GraphEdge {
  source: string
  target: string
  type?: 'contains' | 'calls'
}

interface GraphVizProps {
  nodes: GraphNode[]
  edges: GraphEdge[]
  highlightIds?: { level1: string[]; level2: string[] }
  onNodeClick?: (nodeId: string) => void
}

export function GraphViz({ nodes, edges, highlightIds, onNodeClick }: GraphVizProps) {
  const svgRef = useRef<SVGSVGElement>(null)

  useEffect(() => {
    if (!svgRef.current || nodes.length === 0) return

    const width = svgRef.current.clientWidth || 800
    const height = svgRef.current.clientHeight || 600

    d3.select(svgRef.current).selectAll('*').remove()

    const svg = d3.select(svgRef.current)
      .attr('width', width)
      .attr('height', height)

    // Arrow markers for call edges
    svg.append('defs').selectAll('marker')
      .data(['calls', 'contains'])
      .enter().append('marker')
      .attr('id', d => `arrow-${d}`)
      .attr('viewBox', '0 -5 10 10')
      .attr('refX', 18)
      .attr('refY', 0)
      .attr('markerWidth', 6)
      .attr('markerHeight', 6)
      .attr('orient', 'auto')
      .append('path')
      .attr('d', 'M0,-5L10,0L0,5')
      .attr('fill', d => d === 'calls' ? '#60a5fa' : '#444')

    const simulation = d3.forceSimulation(nodes as any)
      .force('link', d3.forceLink(edges as any).id((d: any) => d.id).distance(120))
      .force('charge', d3.forceManyBody().strength(-400))
      .force('center', d3.forceCenter(width / 2, height / 2))
      .force('collision', d3.forceCollide(20))

    const link = svg.selectAll('line')
      .data(edges)
      .enter().append('line')
      .attr('stroke', (d: any) => d.type === 'calls' ? '#3b82f6' : '#374151')
      .attr('stroke-width', (d: any) => d.type === 'calls' ? 1.5 : 1)
      .attr('stroke-opacity', (d: any) => d.type === 'calls' ? 0.7 : 0.4)
      .attr('marker-end', (d: any) => d.type === 'calls' ? 'url(#arrow-calls)' : null)

    const getNodeColor = (d: any) => {
      if (highlightIds?.level1?.includes(d.id)) return '#ef4444'
      if (highlightIds?.level2?.includes(d.id)) return '#f59e0b'
      if (d.type === 'function' || d.type === 'method') return '#60a5fa'
      if (d.type === 'class' || d.type === 'struct' || d.type === 'interface') return '#34d399'
      if (d.type === 'file') return '#8b5cf6'
      return '#fbbf24'
    }

    const node = svg.selectAll('circle')
      .data(nodes)
      .enter().append('circle')
      .attr('r', (d: any) => d.type === 'file' ? 10 : 7)
      .attr('fill', getNodeColor)
      .attr('stroke', (d: any) => highlightIds?.level1?.includes(d.id) ? '#dc2626' : 'none')
      .attr('stroke-width', 2)
      .call(d3.drag()
        .on('start', (event: any, d: any) => {
          if (!event.active) simulation.alphaTarget(0.3).restart()
          d.fx = d.x; d.fy = d.y
        })
        .on('drag', (event: any, d: any) => { d.fx = event.x; d.fy = event.y })
        .on('end', (event: any, d: any) => {
          if (!event.active) simulation.alphaTarget(0)
          d.fx = null; d.fy = null
        }) as any)
      .on('click', (_: any, d: any) => onNodeClick?.(d.id))

    const labels = svg.selectAll('text')
      .data(nodes)
      .enter().append('text')
      .attr('text-anchor', 'middle')
      .attr('fill', '#cbd5e1')
      .attr('font-size', '10px')
      .attr('pointer-events', 'none')
      .text((d: any) => d.name.substring(0, 12))

    simulation.on('tick', () => {
      link
        .attr('x1', (d: any) => d.source.x)
        .attr('y1', (d: any) => d.source.y)
        .attr('x2', (d: any) => d.target.x)
        .attr('y2', (d: any) => d.target.y)
      node.attr('cx', (d: any) => d.x).attr('cy', (d: any) => d.y)
      labels.attr('x', (d: any) => d.x).attr('y', (d: any) => d.y - 12)
    })

  }, [nodes, edges, highlightIds, onNodeClick])

  return (
    <div className="bg-gray-900 h-full rounded-lg overflow-hidden relative">
      <svg ref={svgRef} className="w-full h-full" />
      {highlightIds && (highlightIds.level1.length > 0 || highlightIds.level2.length > 0) && (
        <div className="absolute bottom-4 left-4 flex gap-3 text-xs">
          <span className="flex items-center gap-1">
            <span className="w-3 h-3 rounded-full bg-red-500 inline-block" />
            Callers directos ({highlightIds.level1.length})
          </span>
          <span className="flex items-center gap-1">
            <span className="w-3 h-3 rounded-full bg-amber-500 inline-block" />
            Nivel 2 ({highlightIds.level2.length})
          </span>
        </div>
      )}
    </div>
  )
}
