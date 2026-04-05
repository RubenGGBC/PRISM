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
}

interface GraphVizProps {
  nodes: GraphNode[]
  edges: GraphEdge[]
  onNodeClick?: (nodeId: string) => void
}

export function GraphViz({ nodes, edges, onNodeClick }: GraphVizProps) {
  const svgRef = useRef<SVGSVGElement>(null)

  useEffect(() => {
    if (!svgRef.current || nodes.length === 0) return

    const width = svgRef.current.clientWidth
    const height = svgRef.current.clientHeight

    // Create simulation
    const simulation = d3
      .forceSimulation(nodes as any)
      .force('link', d3.forceLink(edges as any).id((d: any) => d.id).distance(100))
      .force('charge', d3.forceManyBody().strength(-300))
      .force('center', d3.forceCenter(width / 2, height / 2))

    // Clear previous
    d3.select(svgRef.current).selectAll('*').remove()

    const svg = d3
      .select(svgRef.current)
      .attr('width', width)
      .attr('height', height)

    // Links
    const link = svg
      .selectAll('line')
      .data(edges)
      .enter()
      .append('line')
      .attr('stroke', '#666')
      .attr('stroke-width', 2)

    // Nodes
    const node = svg
      .selectAll('circle')
      .data(nodes)
      .enter()
      .append('circle')
      .attr('r', 8)
      .attr('fill', (d: any) => {
        if (d.type === 'function') return '#60a5fa'
        if (d.type === 'class') return '#34d399'
        return '#fbbf24'
      })
      .call(d3.drag().on('start', dragStarted).on('drag', dragged).on('end', dragEnded) as any)
      .on('click', (_, d: any) => onNodeClick?.(d.id))

    // Labels
    const labels = svg
      .selectAll('text')
      .data(nodes)
      .enter()
      .append('text')
      .attr('x', (d: any) => d.x)
      .attr('y', (d: any) => d.y)
      .attr('text-anchor', 'middle')
      .attr('fill', '#e0e0e0')
      .attr('font-size', '11px')
      .text((d: any) => d.name.substring(0, 10))

    simulation.on('tick', () => {
      link
        .attr('x1', (d: any) => d.source.x)
        .attr('y1', (d: any) => d.source.y)
        .attr('x2', (d: any) => d.target.x)
        .attr('y2', (d: any) => d.target.y)

      node.attr('cx', (d: any) => d.x).attr('cy', (d: any) => d.y)

      labels.attr('x', (d: any) => d.x).attr('y', (d: any) => d.y - 15)
    })

    function dragStarted(event: any, d: any) {
      if (!event.active) simulation.alphaTarget(0.3).restart()
      d.fx = d.x
      d.fy = d.y
    }

    function dragged(event: any, d: any) {
      d.fx = event.x
      d.fy = event.y
    }

    function dragEnded(event: any, d: any) {
      if (!event.active) simulation.alphaTarget(0)
      d.fx = null
      d.fy = null
    }
  }, [nodes, edges, onNodeClick])

  return (
    <div className="bg-gray-900 h-full rounded-lg overflow-hidden">
      <svg ref={svgRef} className="w-full h-full" />
    </div>
  )
}
