import { ReactElement, SVGProps } from 'react'

import { Group } from '@visx/group'
import { LinePath } from '@visx/shape'
import { ScaleLinear, ScaleTime } from 'd3-scale'
import { timeFormat } from 'd3-time-format'

import { Point } from '../types'
import { getDatumValue, isDatumWithValidNumber, SeriesDatum } from '../utils'
import { getPointId } from '../utils/generate-points-field'

import { PointGlyph } from './PointGlyph'

const NULL_LINK = (): undefined => undefined

/**
 * Returns a formatted date text for points aria labels.
 *
 * Example: 2021 January 21 Thursday
 */
const formatXLabel = timeFormat('%d %B %A')

interface LineDataSeriesProps<D> extends SVGProps<SVGGElement> {
    id: string
    xScale: ScaleTime<number, number>
    yScale: ScaleLinear<number, number>
    dataset: SeriesDatum<D>[]
    color: string | undefined
    activePointId?: string
    getLinkURL?: (datum: D, index: number) => string | undefined
    onDatumFocus?: (point: Point) => void
    onDatumClick?: (point: Point) => void
}

export function LineDataSeries<D>(props: LineDataSeriesProps<D>): ReactElement {
    const {
        id,
        xScale,
        yScale,
        dataset,
        color = 'green',
        activePointId,
        tabIndex,
        getLinkURL = NULL_LINK,
        onDatumClick,
        onDatumFocus,
        pointerEvents = 'visiblePainted',
        ...attributes
    } = props

    return (
        <Group tabIndex={tabIndex} pointerEvents={pointerEvents} {...attributes}>
            <LinePath
                data={dataset}
                defined={isDatumWithValidNumber}
                x={data => xScale(data.x)}
                y={data => yScale(getDatumValue(data))}
                stroke={color}
                strokeLinecap="round"
                strokeWidth={2}
                aria-hidden={true}
                pointerEvents="none"
            />

            <Group role="list" pointerEvents={pointerEvents}>
                {dataset.map((datum, index) => {
                    const datumValue = getDatumValue(datum)
                    const link = getLinkURL(datum.datum, index)
                    const pointId = getPointId(id, index)
                    const formattedDate = formatXLabel(datum.x)
                    const datumInfo = { id: pointId, seriesId: id, xValue: datum.x, yValue: datumValue, linkUrl: link }
                    const ariaLabel = link
                        ? `Link point, Y value: ${datumValue}, X value: ${formattedDate}, click to view data point detail`
                        : `Data point, Y value: ${datumValue}, X value: ${formattedDate}`

                    return (
                        <PointGlyph
                            key={pointId}
                            id={pointId}
                            tabIndex={tabIndex}
                            top={yScale(datumValue)}
                            left={xScale(datum.x)}
                            active={activePointId === pointId}
                            color={color}
                            linkURL={link}
                            role="listitem"
                            aria-label={ariaLabel}
                            onFocus={event => onDatumFocus?.({ ...datumInfo, node: event.currentTarget })}
                            onClick={event => {
                                // Stop propagation in order to avoid double call of the onDatumClick
                                // callback (we have click handling here and on the line chart content
                                // level
                                event.stopPropagation()
                                onDatumClick?.({ ...datumInfo, node: event.currentTarget })
                            }}
                        />
                    )
                })}
            </Group>
        </Group>
    )
}
