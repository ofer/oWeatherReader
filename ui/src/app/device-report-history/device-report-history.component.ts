import { Component, Input, OnDestroy, OnInit } from '@angular/core';
import { ApiService } from '../api.service';
import { EChartsOption } from 'echarts';
import { WeatherReport } from '../weather-report';
import { switchMap, takeUntil, Subscription, Subject } from 'rxjs';

@Component({
  selector: 'app-device-report-history',
  templateUrl: './device-report-history.component.html',
  styleUrls: ['./device-report-history.component.scss']
})
export class DeviceReportHistoryComponent implements OnInit, OnDestroy {

  data: DataT[];
  humidityData: DataH[];
  rawData: WeatherReport[];
  private DAYS_OF_HISTORY = 3;

  options: EChartsOption;
  updateOptions: EChartsOption;

  currentModel: string | null = null;
  private subscriptions = new Subscription();
  private destroy$ = new Subject<void>();

  @Input()
  set deviceModel(value: string | null) {
    if (value != null) {
      this.currentModel = value;
      this.loadHistory(value);
    }
  }

  private loadHistory(model: string): void {
    this.api.getHistoricDataForDeviceModel(model).subscribe(historicWeatherReports => {
      this.rawData = historicWeatherReports;
      this.data = this.convertToTemperatureData(historicWeatherReports);
      this.humidityData = this.convertToHumidityData(historicWeatherReports);
      this.updateOptions = {
        series: [
          {
            data: this.data,
          },
          {
            data: this.humidityData
          }
        ],
      };
    });
  }

  convertToTemperatureData(historicWeatherReports: WeatherReport[]): DataT[] {
    return historicWeatherReports.filter(report => this.isReportInRange(report)).map(report => {
      return {
        name: report.Time.toString(),
        value: [report.Time.toString(), report.TemperatureInF]
      } as DataT;
    });
  }

  private findYesterdayTemp(reportTime: Date, reports: WeatherReport[]): number | null {
    const yesterday = new Date(new Date(reportTime).getTime() - 24 * 60 * 60 * 1000);
    let closest: WeatherReport | null = null;
    let closestDiff = Infinity;
    for (const report of reports) {
      if (!this.isReportInRange(report)) continue;
      const reportTimeMs = new Date(report.Time).getTime();
      const diff = Math.abs(reportTimeMs - yesterday.getTime());
      if (diff < closestDiff) {
        closestDiff = diff;
        closest = report;
      }
    }
    return closest !== null && closestDiff < 2 * 60 * 60 * 1000 ? closest.TemperatureInF : null;
  }

  convertToHumidityData(historicWeatherReports: WeatherReport[]): DataH[] {
    return historicWeatherReports.filter(report => this.isReportInRange(report)).map(report => {
      return {
        name: report.Time.toString(),
        value: [report.Time.toString(), report.HumidityInPercentage]
      } as DataH;
    });
  }

  isReportInRange(report: WeatherReport): boolean {
    const reportDate = new Date(report.Time);
    const oldestUseableDate = new Date();
    oldestUseableDate.setDate(oldestUseableDate.getDate() - this.DAYS_OF_HISTORY);
    return reportDate >= oldestUseableDate;
  }

  formatTooltip(params: any[]): string {
    let result = '';
    const tempSeries = params.find((p: any) => p?.seriesName === 'Temperature Data');
    const humiditySeries = params.find((p: any) => p?.seriesName === 'Humidity Data');
    if (tempSeries) {
      const value = tempSeries?.data?.value as number[] | undefined;
      const humidityValue = humiditySeries?.data?.value as number[] | 'unknown';
      if (value == undefined)
      	return 'unknown';
      let time = new Date(value[0]);
      let yesterdayTemp = this.findYesterdayTemp(time, this.rawData);
      if (yesterdayTemp) {
        result = `${value[0]}<br/>Temp: ${value[1]}°F, Humidity ${humidityValue[1]}%<br/>Yesterday Temp°F: ${yesterdayTemp}°F`;
      } else {
        result = `${value?.[0] ?? ''}<br/>Temp: ${value?.[1] ?? ''}°F Humidity ${humidityValue[1]}%`;
      }
    }
    return result;
  }

  constructor(private api: ApiService) {
    this.data = [];
    this.humidityData = [];
    this.rawData = [];

    // initialize chart options:
    this.options = {
      title: {
        text: 'Weather Report History',
      },
      tooltip: {
        trigger: 'axis',
        axisPointer: {
          animation: false,
        },
        formatter: this.formatTooltip.bind(this),
      } as any,
      xAxis: {
        type: 'time',
        splitLine: {
          show: false,
        },
      },
      yAxis: {
        type: 'value',
        scale: true,
        splitLine: {
          show: false,
        },
      },
      series: [
        {
          name: 'Temperature Data',
          type: 'line',
          showSymbol: false,
          data: this.data as any,
        },
        {
          name: 'Humidity Data',
          type: 'line',
          showSymbol: false,
          data: this.humidityData,
        }
      ],
    };
    this.updateOptions = {
      series: [
        {
          data: this.data,
        },
        {
          data: this.humidityData
        }
      ],
    };

  }

  ngOnInit(): void {
    const eventSubscription = this.api.weatherReportEvents$
      .pipe(
        switchMap(event => {
          if (event.deviceModel === this.currentModel) {
            return this.api.getHistoricDataForDeviceModel(this.currentModel!);
          }
          return [];
        }),
        takeUntil(this.destroy$)
      )
      .subscribe(historicWeatherReports => {
        this.data = this.convertToTemperatureData(historicWeatherReports);
        this.humidityData = this.convertToHumidityData(historicWeatherReports);
        this.updateOptions = {
          series: [
            {
              data: this.data,
            },
            {
              data: this.humidityData
            }
          ],
        };
      });

    this.subscriptions.add(eventSubscription);
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
    this.subscriptions.unsubscribe();
  }
}

type DataT = {
  name: string;
  value: [string, number];
};

type DataH = {
  name: string;
  value: [string, number];
};
