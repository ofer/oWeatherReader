import { Component, Input } from '@angular/core';
import { ApiService } from '../api.service';
import { EChartsOption } from 'echarts';
import { WeatherReport } from '../weather-report';

@Component({
  selector: 'app-device-report-history',
  templateUrl: './device-report-history.component.html',
  styleUrls: ['./device-report-history.component.scss']
})
export class DeviceReportHistoryComponent {

  // @Input() deviceModel!: string;

  @Input()
  set deviceModel(value: string) {
    // Call API to get data for the new device model
    this.api.getHistoricDataForDeviceModel(value).subscribe(historicWeatherReports => {
      // Update chart data with new data
      this.data = this.convertToData(historicWeatherReports);
      this.updateOptions = {
        series: [
          {
            data: this.data,
          },
        ],
      };
    });
  }

  convertToData(historicWeatherReports: WeatherReport[]): DataT[] {
    return historicWeatherReports.map(report => {
      return {
        name: report.Time.toString(),
        value: [report.Time.toString(), report.TemperatureInF] } as DataT;
    });
  }


  private data: DataT[];
  options: EChartsOption;
  updateOptions: EChartsOption;


  constructor(private api: ApiService) {
    this.data = [];
    // initialize chart options:
    this.options = {
      title: {
        text: 'Weather Report History',
      },
      tooltip: {
        trigger: 'axis',
        // formatter: params => {
        //   params = params[0];
        //   const date = new Date(params.name);
        //   return (
        //     date.getDate() +
        //     '/' +
        //     (date.getMonth() + 1) +
        //     '/' +
        //     date.getFullYear() +
        //     ' : ' +
        //     params.value[1]
        //   );
        // },
        axisPointer: {
          animation: false,
        },
      },
      xAxis: {
        type: 'time',
        splitLine: {
          show: false,
        },
      },
      yAxis: {
        type: 'value',
        boundaryGap: [0, '100%'],
        splitLine: {
          show: false,
        },
      },
      series: [
        {
          name: 'Temperature Data',
          type: 'line',
          showSymbol: false,
          data: this.data,
        },
      ],
    };
    this.updateOptions = {
      series: [
        {
          data: this.data,
        },
      ],
    };

  }

  ngOnInit(): void {
    // // Mock dynamic data:
    // this.timer = setInterval(() => {
    //   for (let i = 0; i < 5; i++) {
    //     this.data.shift();
    //     this.data.push(this.randomData());
    //   }

    //   // update series data:
    // }, 1000);
  }
}

type DataT = {
  name: string;
  value: [string, number];
};
