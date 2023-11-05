import { Injectable } from '@angular/core';
import { ApiService } from './api.service';

@Injectable({
  providedIn: 'root'
})
export class SettingsService {

  private MONITORING_DEVICE_NAME_KEY = 'monitoringDeviceNameKey';

  getMonitoringDeviceName(): string | null {
    let montoringDeviceName = localStorage.getItem(this.MONITORING_DEVICE_NAME_KEY);
    return montoringDeviceName;
  }

  setMonitoringDeviceName(deviceName: string) {
    localStorage.setItem(this.MONITORING_DEVICE_NAME_KEY, deviceName);
  }

  constructor(api: ApiService) { }
}
