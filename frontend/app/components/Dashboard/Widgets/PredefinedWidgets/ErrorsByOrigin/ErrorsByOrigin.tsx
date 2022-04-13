import React from 'react';
import { NoContent } from 'UI';
import { Styles } from '../../common';
import { 
    BarChart, Bar, CartesianGrid, Tooltip,
    LineChart, Line, Legend, ResponsiveContainer, 
    XAxis, YAxis
  } from 'recharts';

interface Props {
    data: any
}
function ErrorsByOrigin(props: Props) {
    const { data } = props;
    return (
        <NoContent
          size="small"
          show={ data.chart.length === 0 }
        >
          <ResponsiveContainer height={ 240 } width="100%">
            <BarChart
              data={data.chart}
              margin={Styles.chartMargins}
              syncId="errorsPerType"
            //   syncId={ showSync ? "errorsPerType" : undefined }
            >
              <CartesianGrid strokeDasharray="3 3" vertical={ false } stroke="#EEEEEE" />
              <XAxis
                {...Styles.xaxis}
                dataKey="time"
                // interval={params.density/7}
              />
              <YAxis
                {...Styles.yaxis}
                label={{ ...Styles.axisLabelLeft, value: "Number of Errors" }}
                allowDecimals={false}
              />
              <Legend />
              <Tooltip {...Styles.tooltip} />
              <Bar minPointSize={1} name={<span className="float">1<sup>st</sup> Party</span>} dataKey="firstParty" stackId="a" fill={Styles.colors[0]} />
              <Bar name={<span className="float">3<sup>rd</sup> Party</span>} dataKey="thirdParty" stackId="a" fill={Styles.colors[2]} />
            </BarChart>
          </ResponsiveContainer>
        </NoContent>
    );
}

export default ErrorsByOrigin;