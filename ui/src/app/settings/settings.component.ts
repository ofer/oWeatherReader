import { Component } from '@angular/core';
import { ApiService } from '../api.service';
import { SettingsService } from '../settings.service';
import { DeviceModel } from '../device-model';

@Component({
  selector: 'app-settings',
  templateUrl: './settings.component.html',
  styleUrls: ['./settings.component.scss']
})
export class SettingsComponent {

  deviceModels: DeviceModel[];

  public get selectedDeviceModel(): string | null {
    return this.settingsService.getMonitoringDeviceName();
  }

  public set selectedDeviceModel(value: string) { 
    this.settingsService.setMonitoringDeviceName(value);
  }

  constructor(private api: ApiService, private settingsService: SettingsService) {
    this.deviceModels = [];
    this.api.getModels().subscribe(deviceModels => {
      this.deviceModels = deviceModels;
    });
  }
}
