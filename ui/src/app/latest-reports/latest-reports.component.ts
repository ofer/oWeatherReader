import { Component, inject } from '@angular/core';
import { Breakpoints, BreakpointObserver } from '@angular/cdk/layout';
import { map } from 'rxjs/operators';
import { Observable } from 'rxjs';
import { ApiService } from '../api.service';
import { SettingsService } from '../settings.service';
import { WeatherReport } from '../weather-report';

@Component({
  selector: 'app-latest-reports',
  templateUrl: './latest-reports.component.html',
  styleUrls: ['./latest-reports.component.scss']
})
export class LatestReportsComponent {
  private breakpointObserver = inject(BreakpointObserver);

  latestReport: Observable<WeatherReport>;
  deviceModelNames: string[] | null;

  /** Based on the screen size, switch from standard to one column per row */
  // cards = this.breakpointObserver.observe(Breakpoints.Handset).pipe(
  //   map(({ matches }) => {
  //     if (matches) {
  //       return [
  //         { title: 'Card 1', cols: 2, rows: 1 },
  //         // { title: 'Card 2', cols: 1, ro  ws: 1 },
  //         // { title: 'Card 3', cols: 1, rows: 1 },
  //         // { title: 'Card 4', cols: 1, rows: 1 }
  //       ];
  //     }

  //     return [
  //       { title: 'Card 1', cols: 2, rows: 1 },
  //       // { title: 'Card 2', cols: 1, rows: 1 },
  //       // { title: 'Card 3', cols: 1, rows: 2 },
  //       // { title: 'Card 4', cols: 1, rows: 1 }
  //     ];
  //   })
  // );

  constructor(private apiService: ApiService, settingsService: SettingsService) {
    this.latestReport = apiService.latestReportObserver;
    this.deviceModelNames = settingsService.getMonitoringDeviceNames();
  }
}
