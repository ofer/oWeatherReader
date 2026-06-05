import { Component, OnInit, OnDestroy } from '@angular/core';
import { ApiService, DailyAgg } from '../api.service';
import { SettingsService } from '../settings.service';
import { EChartsOption } from 'echarts';
import { Subscription } from 'rxjs';

interface ComfortLevel {
  date: string;
  model: string;
  modelName: string;
  rating: string;
  color: string;
  icon: string;
}

interface ComfortScaleItem {
  label: string;
  range: string;
  color: string;
  icon: string;
}

@Component({
  selector: 'app-history-page',
  templateUrl: './history-page.component.html',
  styleUrls: ['./history-page.component.scss']
})
export class HistoryPageComponent implements OnInit, OnDestroy {
  comfortScale: ComfortScaleItem[] = [
    { label: 'Very Cool', range: 'HI < 60°F', color: '#4fc3f7', icon: 'wb_sunny' },
    { label: 'Cool', range: 'HI 60–69°F', color: '#29b6f6', icon: 'wb_sunny' },
    { label: 'Comfortable', range: 'HI 70–79°F', color: '#66bb6a', icon: 'thumb_up' },
    { label: 'Warm', range: 'HI 80–84°F', color: '#ffa726', icon: 'local_fire_department' },
    { label: 'Warm / Humid', range: 'HI 85–89°F', color: '#ff7043', icon: 'humidity_high' },
    { label: 'Hot / Humid', range: 'HI 90–99°F', color: '#ef5350', icon: 'thermostat' },
    { label: 'Dangerous', range: 'HI ≥ 100°F', color: '#c62828', icon: 'warning' },
  ];

  monitoringIndoor: string[] = [];
  monitoringOutdoor: string[] = [];
  daysRange: number = 90;
  loading: boolean = false;
  isMonthly: boolean = false;

  indoorAggregates: DailyAgg[] = [];
  outdoorAggregates: DailyAgg[] = [];

  indoorComfortLevels: ComfortLevel[] = [];
  outdoorComfortLevels: ComfortLevel[] = [];

  indoorTempChartOption: EChartsOption = {};
  outdoorTempChartOption: EChartsOption = {};
  indoorHumidityChartOption: EChartsOption = {};
  outdoorHumidityChartOption: EChartsOption = {};

  private subscriptions = new Subscription();

  constructor(
    private api: ApiService,
    private settings: SettingsService
  ) {}

  ngOnInit(): void {
    const indoorModel = this.settings.getIndoorDeviceModel();
    if (indoorModel) {
      this.monitoringIndoor = [indoorModel];
    }

    const outdoorModel = this.settings.getOutdoorDeviceModel();
    if (outdoorModel) {
      this.monitoringOutdoor = [outdoorModel];
    }

    this.loadHistory();
  }

  onDaysChange(): void {
    this.loadHistory();
  }

  loadHistory(): void {
    this.loading = true;
    this.isMonthly = this.daysRange > 90;

    const indoorSub = this.api.getDailyAggregates(this.monitoringIndoor, this.daysRange).subscribe(
      (data: DailyAgg[]) => {
        this.indoorAggregates = data;
        this.indoorComfortLevels = this.calculateComfortLevels(data);
        this.indoorTempChartOption = this.buildTempChartFor(data, 'Indoor');
        this.indoorHumidityChartOption = this.buildHumidityChartFor(data, 'Indoor');
        this.loading = this.indoorAggregates.length === 0 || this.checkOutdoorLoaded();
      },
      () => { this.loading = this.checkOutdoorLoaded(); }
    );

    const outdoorSub = this.api.getDailyAggregates(this.monitoringOutdoor, this.daysRange).subscribe(
      (data: DailyAgg[]) => {
        this.outdoorAggregates = data;
        this.outdoorComfortLevels = this.calculateComfortLevels(data);
        this.outdoorTempChartOption = this.buildTempChartFor(data, 'Outdoor');
        this.outdoorHumidityChartOption = this.buildHumidityChartFor(data, 'Outdoor');
        this.loading = false;
      },
      () => {
        this.loading = false;
      }
    );

    this.subscriptions.add(indoorSub);
    this.subscriptions.add(outdoorSub);
  }

  private checkOutdoorLoaded(): boolean {
    return this.monitoringOutdoor.length > 0;
  }

  calculateComfortLevels(aggregates: DailyAgg[]): ComfortLevel[] {
    return [...aggregates]
      .sort((a, b) => b.date.localeCompare(a.date))
      .map(a => {
        const { rating, color, icon } = this.getComfortRating(a.avgTemp, a.avgHumidity);
        return {
          date: a.date,
          model: a.model,
          modelName: a.modelName,
          rating,
          color,
          icon
        };
      });
  }

  getComfortRating(tempF: number, humidity: number): { rating: string; color: string; icon: string } {
    const heatIndex = this.calculateHeatIndex(tempF, humidity);

    if (heatIndex < 60) return { rating: 'Very Cool', color: '#4fc3f7', icon: 'wb_sunny' };
    if (heatIndex < 70) return { rating: 'Cool', color: '#29b6f6', icon: 'wb_sunny' };
    if (heatIndex < 80) return { rating: 'Comfortable', color: '#66bb6a', icon: 'thumb_up' };
    if (heatIndex < 85) return { rating: 'Warm', color: '#ffa726', icon: 'local_fire_department' };
    if (heatIndex < 90) return { rating: 'Warm / Humid', color: '#ff7043', icon: 'humidity_high' };
    if (heatIndex < 100) return { rating: 'Hot / Humid', color: '#ef5350', icon: 'thermostat' };
    return { rating: 'Dangerous', color: '#c62828', icon: 'warning' };
  }

  calculateHeatIndex(tempF: number, humidity: number): number {
    if (tempF < 80) return tempF;
    const T = tempF;
    const RH = Math.min(humidity, 100);
    let hi = (-42.379 + 2.04901523 * T + 10.14333127 * RH
      - 0.22475541 * T * RH - 0.00683783 * T * T
      - 0.05481717 * RH * RH
      + 0.00122874 * T * T * RH + 0.00085282 * T * RH * RH
      - 0.00000199 * T * T * RH * RH);
    if (RH < 13 && T >= 80 && T <= 112) {
      hi -= ((13 - RH) / 4) * Math.sqrt((17 - Math.abs(T - 95)) / 17);
    }
    if (RH > 85 && T >= 80 && T <= 87) {
      hi += ((RH - 85) / 10) * ((87 - T) / 5);
    }
    return hi;
  }

  private buildTempChartFor(aggregates: DailyAgg[], label: 'Indoor' | 'Outdoor'): EChartsOption {
    if (aggregates.length === 0) return {};

    const sorted = [...aggregates].sort((a, b) => a.date.localeCompare(b.date));
    const dates = sorted.map(a => a.date);
    const avg = this.fillDataPoints(sorted, dates);
    const high = this.fillHighPoints(sorted, dates);
    const low = this.fillLowPoints(sorted, dates);

    const avgHigh = this.fillAvgHighPoints(sorted, dates);
    const avgLow = this.fillAvgLowPoints(sorted, dates);

    const colors = label === 'Indoor'
      ? { avg: '#1976d2', high: '#64b5f6', low: '#90caf9', avgHigh: '#1565c0', avgLow: '#42a5f5' }
      : { avg: '#d32f2f', high: '#ef5350', low: '#ef9a9a', avgHigh: '#b71c1c', avgLow: '#e57373' };

    const series: any[] = [
      { name: `${label} High`,       type: 'line', data: high,   smooth: true, symbol: 'none', lineStyle: { width: 1.5, type: 'dashed' },                    color: colors.high },
      { name: `${label} Avg High`,   type: 'line', data: avgHigh, smooth: true, symbol: 'none', lineStyle: { width: 1.5, type: 'dashed' }, dashOffset: [5, 5], color: colors.avgHigh },
      { name: `${label} Avg`,        type: 'line', data: avg,    smooth: true, symbol: 'none', lineStyle: { width: 2 },                                       color: colors.avg },
      { name: `${label} Avg Low`,    type: 'line', data: avgLow,  smooth: true, symbol: 'none', lineStyle: { width: 1.5, type: 'dotted' },  dashOffset: [5, 5], color: colors.avgLow },
      { name: `${label} Low`,        type: 'line', data: low,    smooth: true, symbol: 'none', lineStyle: { width: 1.5, type: 'dotted' },                     color: colors.low },
    ];

    return {
      title: { text: `${label} ${this.isMonthly ? 'Monthly' : 'Daily'} Temperature (°F)`, left: 'center', textStyle: { fontSize: 16 } },
      tooltip: {
        trigger: 'axis',
        formatter: (params: any[]) => {
          let s = `<b>${params[0]?.axisValue}</b><br/>`;
          params.forEach((p: any) => { s += `${p.marker}${p.seriesName}: ${p.value}°F<br/>`; });
          return s;
        }
      },
      legend: { data: series.map(s => s.name), bottom: 0 },
      grid: { left: 50, right: 30, top: 60, bottom: 50 },
      xAxis: { type: 'category', data: dates, axisLabel: { rotate: 45 } },
      yAxis: { type: 'value', name: '°F', splitLine: { lineStyle: { type: 'dashed' } } },
      series
    } as EChartsOption;
  }

  private buildHumidityChartFor(aggregates: DailyAgg[], label: 'Indoor' | 'Outdoor'): EChartsOption {
    if (aggregates.length === 0) return {};

    const sorted = [...aggregates].sort((a, b) => a.date.localeCompare(b.date));
    const dates = sorted.map(a => a.date);
    const avgMap = new Map<string, number>();
    const highMap = new Map<string, number>();
    const lowMap = new Map<string, number>();
    const avgHighMap = new Map<string, number>();
    const avgLowMap = new Map<string, number>();
    sorted.forEach(a => {
      avgMap.set(a.date, Math.round(a.avgHumidity));
      highMap.set(a.date, Math.round(a.highHumidity));
      lowMap.set(a.date, Math.round(a.lowHumidity));
      avgHighMap.set(a.date, Math.round(a.avgHighHumidity ?? 0));
      avgLowMap.set(a.date, Math.round(a.avgLowHumidity ?? 0));
    });
    const avg = dates.map(d => avgMap.get(d) ?? null);
    const high = dates.map(d => highMap.get(d) ?? null);
    const low = dates.map(d => lowMap.get(d) ?? null);
    const avgHigh = dates.map(d => avgHighMap.get(d) ?? null);
    const avgLow = dates.map(d => avgLowMap.get(d) ?? null);

    const colors = label === 'Indoor'
      ? { avg: '#1976d2', high: '#64b5f6', low: '#90caf9', avgHigh: '#1565c0', avgLow: '#42a5f5' }
      : { avg: '#d32f2f', high: '#ef5350', low: '#ef9a9a', avgHigh: '#b71c1c', avgLow: '#e57373' };

    const series: any[] = [
      { name: `${label} Avg`,     type: 'line', data: avg,     smooth: true, symbol: 'none', lineStyle: { width: 2 },                           color: colors.avg },
      { name: `${label} Avg High`, type: 'line', data: avgHigh, smooth: true, symbol: 'none', lineStyle: { width: 1.5, type: 'dashed' },  dashOffset: [5, 5], color: colors.avgHigh },
      { name: `${label} High`,    type: 'line', data: high,    smooth: true, symbol: 'none', lineStyle: { width: 1.5, type: 'dashed' },                       color: colors.high },
      { name: `${label} Avg Low`,  type: 'line', data: avgLow,  smooth: true, symbol: 'none', lineStyle: { width: 1.5, type: 'dotted' },  dashOffset: [5, 5], color: colors.avgLow },
      { name: `${label} Low`,     type: 'line', data: low,     smooth: true, symbol: 'none', lineStyle: { width: 1.5, type: 'dotted' },                       color: colors.low },
    ];

    return {
      title: { text: `${label} ${this.isMonthly ? 'Monthly' : 'Daily'} Humidity (%)`, left: 'center', textStyle: { fontSize: 16 } },
      tooltip: {
        trigger: 'axis',
        formatter: (params: any[]) => {
          let s = `<b>${params[0]?.axisValue}</b><br/>`;
          params.forEach((p: any) => { s += `${p.marker}${p.seriesName}: ${p.value}%<br/>`; });
          return s;
        }
      },
      legend: { data: series.map(s => s.name), bottom: 0 },
      grid: { left: 50, right: 30, top: 60, bottom: 50 },
      xAxis: { type: 'category', data: dates, axisLabel: { rotate: 45 } },
      yAxis: { type: 'value', name: '%', min: 0, max: 100, splitLine: { lineStyle: { type: 'dashed' } } },
      series
    } as EChartsOption;
  }

  private fillDataPoints(aggregates: DailyAgg[], dates: string[]): (number | null)[] {
    const map = new Map<string, number>();
    aggregates.forEach(a => { map.set(a.date, Math.round(a.avgTemp * 10) / 10); });
    return dates.map(d => map.get(d) ?? null);
  }

  private fillHighPoints(aggregates: DailyAgg[], dates: string[]): (number | null)[] {
    const map = new Map<string, number>();
    aggregates.forEach(a => { map.set(a.date, Math.round(a.highTemp * 10) / 10); });
    return dates.map(d => map.get(d) ?? null);
  }

  private fillLowPoints(aggregates: DailyAgg[], dates: string[]): (number | null)[] {
    const map = new Map<string, number>();
    aggregates.forEach(a => { map.set(a.date, Math.round(a.lowTemp * 10) / 10); });
    return dates.map(d => map.get(d) ?? null);
  }

  private fillAvgHighPoints(aggregates: DailyAgg[], dates: string[]): (number | null)[] {
    const map = new Map<string, number>();
    aggregates.forEach(a => { map.set(a.date, Math.round((a.avgHighTemp ?? 0) * 10) / 10); });
    return dates.map(d => map.get(d) ?? null);
  }

  private fillAvgLowPoints(aggregates: DailyAgg[], dates: string[]): (number | null)[] {
    const map = new Map<string, number>();
    aggregates.forEach(a => { map.set(a.date, Math.round((a.avgLowTemp ?? 0) * 10) / 10); });
    return dates.map(d => map.get(d) ?? null);
  }

  private fillAvgHighHumidityPoints(aggregates: DailyAgg[], dates: string[]): (number | null)[] {
    const map = new Map<string, number>();
    aggregates.forEach(a => { map.set(a.date, Math.round(a.avgHighHumidity ?? 0)); });
    return dates.map(d => map.get(d) ?? null);
  }

  private fillAvgLowHumidityPoints(aggregates: DailyAgg[], dates: string[]): (number | null)[] {
    const map = new Map<string, number>();
    aggregates.forEach(a => { map.set(a.date, Math.round(a.avgLowHumidity ?? 0)); });
    return dates.map(d => map.get(d) ?? null);
  }

  getComfortForAgg(agg: DailyAgg): { rating: string; color: string; icon: string } {
    return this.getComfortRating(agg.avgTemp, agg.avgHumidity);
  }

  getComfortGroups(aggregates: DailyAgg[]): { name: string; color: string; count: number }[] {
    const levels = this.calculateComfortLevels(aggregates);
    const counts = new Map<string, number>();
    levels.forEach(l => {
      counts.set(l.rating, (counts.get(l.rating) || 0) + 1);
    });
    return Array.from(counts.entries()).map(([name, count]) => {
      const level = levels.find(l => l.rating === name);
      return { name, color: level?.color || '#999', count };
    });
  }

  get hasIndoorData(): boolean { return this.indoorAggregates.length > 0; }
  get hasOutdoorData(): boolean { return this.outdoorAggregates.length > 0; }
  get hasAnyData(): boolean { return this.hasIndoorData || this.hasOutdoorData; }
  get aggregates(): DailyAgg[] { return [...this.indoorAggregates, ...this.outdoorAggregates]; }
  get recordCount(): number { return this.indoorAggregates.length + this.outdoorAggregates.length; }

  displayedColumns: string[] = ['date', 'model', 'avgTemp', 'highTemp', 'lowTemp', 'avgHumidity', 'comfort'];

  ngOnDestroy(): void {
    this.subscriptions.unsubscribe();
  }
}
