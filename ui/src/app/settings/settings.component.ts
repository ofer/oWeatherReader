import { Component, OnInit } from '@angular/core';
import { ApiService } from '../api.service';
import { SettingsService } from '../settings.service';
import { DeviceModel } from '../device-model';

export type DeviceListItem = {
  name: string;
  deviceModel: string;
  reportCount: number;
};

@Component({
  selector: 'app-settings',
  templateUrl: './settings.component.html',
  styleUrls: ['./settings.component.scss']
})
export class SettingsComponent implements OnInit {
  devices: DeviceListItem[] = [];
  selectedIndoor: string | null = null;
  selectedOutdoor: string | null = null;

  constructor(
    private api: ApiService,
    private settings: SettingsService
  ) {}

  ngOnInit(): void {
    this.api.getModels().subscribe(deviceModels => {
      deviceModels.forEach(element => {
        this.devices.push({
          name: element.Name,
          deviceModel: element.DeviceModel,
          reportCount: element.ReportCount
        });
      });
      this.selectedIndoor = this.settings.getIndoorDeviceModel();
      this.selectedOutdoor = this.settings.getOutdoorDeviceModel();
    });
  }

  onIndoorChange(deviceModel: string): void {
    this.settings.setIndoorDeviceModel(deviceModel);
  }

  onOutdoorChange(deviceModel: string): void {
    this.settings.setOutdoorDeviceModel(deviceModel);
  }

  isDisabledForOtherSensor(deviceModel: string): boolean {
    if (this.selectedIndoor && this.selectedIndoor === deviceModel && !this.selectedOutdoor) {
      return true;
    }
    if (this.selectedOutdoor && this.selectedOutdoor === deviceModel && !this.selectedIndoor) {
      return true;
    }
    return false;
  }
}
