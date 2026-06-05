import { Injectable } from '@angular/core';

@Injectable({
  providedIn: 'root'
})
export class SettingsService {

  private INDOOR_DEVICE_MODEL_KEY = 'indoorDeviceModel';
  private OUTDOOR_DEVICE_MODEL_KEY = 'outdoorDeviceModel';

  getIndoorDeviceModel(): string | null {
    return localStorage.getItem(this.INDOOR_DEVICE_MODEL_KEY);
  }

  setIndoorDeviceModel(deviceModel: string): void {
    localStorage.setItem(this.INDOOR_DEVICE_MODEL_KEY, deviceModel);
  }

  getOutdoorDeviceModel(): string | null {
    return localStorage.getItem(this.OUTDOOR_DEVICE_MODEL_KEY);
  }

  setOutdoorDeviceModel(deviceModel: string): void {
    localStorage.setItem(this.OUTDOOR_DEVICE_MODEL_KEY, deviceModel);
  }

  constructor() {
  }
}
