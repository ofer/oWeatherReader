import { Injectable } from '@angular/core';
import { ApiService } from './api.service';
import { SettingsService } from './settings.service';
import { WeatherReport } from './weather-report';

@Injectable({
  providedIn: 'root'
})
export class LatestWeatherReporterService {

  // private _monitoredDevices: string[] | null;
  private _monitoredDevices: { deviceModel: string; latestReport: WeatherReport | null }[] | null;

  constructor(private _api: ApiService, private _settings: SettingsService) {
    this._monitoredDevices = _settings.getMonitoringDeviceNames()?.map((deviceModel) => { return { deviceModel, latestReport: null } }) || [];

    _api.latestReportObserver.subscribe((report) => {
      let montitoredDevice = this._monitoredDevices?.find((device) => device.deviceModel === report.DeviceModel);
      if (montitoredDevice) {
        montitoredDevice.latestReport = report;
        console.log(`Latest report for ${report.DeviceModel}: ${report.TemperatureInF}`);
      }
    });
  }

  get latestWeatherReports() : WeatherReport[] {
    return this._monitoredDevices?.map(md => md.latestReport).filter(r => r !== null) as WeatherReport[];
  }
}
