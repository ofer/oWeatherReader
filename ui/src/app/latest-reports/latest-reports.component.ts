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


  constructor(private apiService: ApiService, settingsService: SettingsService) {
    this.latestReport = apiService.latestReportObserver;
    const indoor = settingsService.getIndoorDeviceModel();
    const outdoor = settingsService.getOutdoorDeviceModel();
    const models: string[] = [];
    if (indoor) models.push(indoor);
    if (outdoor) models.push(outdoor);
    this.deviceModelNames = models.length > 0 ? models : null;
  }
}
