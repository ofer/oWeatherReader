import { Injectable } from '@angular/core';
import { ApiService } from './api.service';

@Injectable({
  providedIn: 'root'
})
export class SettingsService {

  private MONITORING_DEVICE_NAME_KEY = 'monitoringDeviceNameKey';

  getMonitoringDeviceNames(): string[] | null {
    var monitoringDevicesString = localStorage.getItem(this.MONITORING_DEVICE_NAME_KEY);
    if (monitoringDevicesString == null) {
      return null;
    }
    try {
      let montoringDeviceNames = JSON.parse(monitoringDevicesString);
      return montoringDeviceNames;
    }
    catch (e) {
      console.log(e);
      return null;
    }
  }

  setMonitoringDeviceName(deviceName: string, shouldMonitor: boolean) {
    let montoringDeviceNames = this.getMonitoringDeviceNames();
    if (montoringDeviceNames == null) {
      montoringDeviceNames = [];
    }
    if (shouldMonitor) {
      montoringDeviceNames.push(deviceName);
    } else {
      montoringDeviceNames = montoringDeviceNames.filter(d => d != deviceName);
    }
    localStorage.setItem(this.MONITORING_DEVICE_NAME_KEY, JSON.stringify(montoringDeviceNames));
  }

  constructor() {
  }
}
